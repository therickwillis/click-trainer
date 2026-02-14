#!/usr/bin/env python3
"""Integration test: full game lifecycle with DB verification.

Prerequisites:
  - PostgreSQL running (e.g. via devcontainer docker-compose)
  - App running with DATABASE_URL set and ROUND_DURATION=5
  - playwright-cli available and initialized

Usage:
  APP_URL=http://localhost:8080 \
  DATABASE_URL=postgres://clicktrainer:clicktrainer@db:5432/clicktrainer?sslmode=disable \
  ROUND_DURATION=5 \
  python3 scripts/integration_test.py
"""

import os
import re
import subprocess
import sys
import time

APP_URL = os.environ.get("APP_URL", "http://localhost:8080")
DB_URL = os.environ.get("DATABASE_URL", "postgres://clicktrainer:clicktrainer@db:5432/clicktrainer?sslmode=disable")
ROUND_DURATION = int(os.environ.get("ROUND_DURATION", "5"))

passed = 0
failed = 0


def run(cmd: str, timeout: int = 10) -> str:
    result = subprocess.run(cmd, shell=True, capture_output=True, text=True, timeout=timeout)
    return result.stdout + result.stderr


def sql(query: str) -> str:
    return run(f'psql "{DB_URL}" -tAc "{query}"').strip()


def pw(session: str, *args: str) -> str:
    cmd = f"playwright-cli -s={session} " + " ".join(args)
    return run(cmd, timeout=15)


def pw_eval(session: str, js: str) -> str:
    """Run JS via playwright-cli eval. Returns the result line."""
    result = subprocess.run(
        ["playwright-cli", f"-s={session}", "eval", js],
        capture_output=True, text=True, timeout=15,
    )
    out = result.stdout + result.stderr
    # Extract the "### Result" value
    for line in out.splitlines():
        line = line.strip()
        if line and not line.startswith("#") and not line.startswith("```") and not line.startswith("await"):
            return line
    return out.strip()


def snapshot_refs(text: str) -> dict[str, str]:
    """Parse snapshot YAML into {ref: description} dict."""
    refs = {}
    for line in text.splitlines():
        m = re.search(r'\[ref=(\w+)\]', line)
        if m:
            refs[m.group(1)] = line.strip()
    return refs


def find_ref(refs: dict[str, str], *keywords: str) -> str | None:
    """Find first ref whose description contains all keywords (case-insensitive)."""
    for ref, desc in refs.items():
        lower = desc.lower()
        if all(k.lower() in lower for k in keywords):
            return ref
    return None


def assert_ge(label: str, minimum: int, actual: int):
    global passed, failed
    if actual >= minimum:
        print(f"  PASS: {label} (got {actual} >= {minimum})")
        passed += 1
    else:
        print(f"  FAIL: {label} — expected >= {minimum}, got {actual}")
        failed += 1


def assert_gt(label: str, minimum: int, actual: int):
    global passed, failed
    if actual > minimum:
        print(f"  PASS: {label} (got {actual} > {minimum})")
        passed += 1
    else:
        print(f"  FAIL: {label} — expected > {minimum}, got {actual}")
        failed += 1


def snap(session: str) -> dict[str, str]:
    """Take a snapshot and return parsed refs from the YAML file."""
    out = pw(session, "snapshot")
    # Output contains a line like: - [Snapshot](.playwright-cli/page-TIMESTAMP.yml)
    m = re.search(r'\((.+?\.yml)\)', out)
    if not m:
        return {}
    yml_path = os.path.join("/workspace", m.group(1))
    try:
        with open(yml_path) as f:
            return snapshot_refs(f.read())
    except FileNotFoundError:
        return {}


def cleanup_sessions():
    for s in ("p1", "p2"):
        pw(s, "close")


# ------------------------------------------------------------------
print("=== Integration Test: Full Game Lifecycle ===\n")

# Step 0: Connectivity
print("--- Step 0: Verify connectivity ---")
health = run(f"curl -sf {APP_URL}/health")
if '"status":"ok"' not in health:
    print(f"  ERROR: App not reachable ({health})")
    sys.exit(1)
print("  App is up.")
if not sql("SELECT 1"):
    print("  ERROR: Cannot connect to DB")
    sys.exit(1)
print("  DB is up.")

# Clean DB
sql("DELETE FROM player_badges")
sql("DELETE FROM click_events")
sql("DELETE FROM game_players")
sql("DELETE FROM games")
sql("DELETE FROM players")
print("  DB cleaned.")

# ------------------------------------------------------------------
# Step 1: Player 1 creates a room
print("\n--- Step 1: Player 1 creates a room ---")
pw("p1", "open", APP_URL)
time.sleep(1)
refs = snap("p1")
create_ref = find_ref(refs, "Create Room")
if not create_ref:
    print("  ERROR: Could not find Create Room button")
    print("  Refs:", refs)
    cleanup_sessions()
    sys.exit(1)
pw("p1", "click", create_ref)
time.sleep(1)

# ------------------------------------------------------------------
# Step 2: Player 1 registers
print("\n--- Step 2: Player 1 registers as Alice ---")
refs = snap("p1")
name_ref = find_ref(refs, "textbox")
if not name_ref:
    print("  ERROR: Could not find name input")
    print("  Refs:", refs)
    cleanup_sessions()
    sys.exit(1)
pw("p1", "fill", name_ref, '"Alice"')
time.sleep(0.5)

refs = snap("p1")
# Look for a submit/join/enter button (not "Join Room" on home page — this is the register form)
submit_ref = find_ref(refs, "button", "Join") or find_ref(refs, "button", "Enter") or find_ref(refs, "button", "Go") or find_ref(refs, "button", "Submit")
if not submit_ref:
    # Fall back to any button that isn't "Create Room"
    for ref, desc in refs.items():
        if "button" in desc.lower() and "create" not in desc.lower():
            submit_ref = ref
            break
if not submit_ref:
    print("  ERROR: Could not find submit button")
    print("  Refs:", refs)
    cleanup_sessions()
    sys.exit(1)
pw("p1", "click", submit_ref)
time.sleep(1)

# Get room code from the snapshot YAML
room_code = None
out = pw("p1", "snapshot")
m = re.search(r'\((.+?\.yml)\)', out)
if m:
    yml_path = os.path.join("/workspace", m.group(1))
    try:
        with open(yml_path) as f:
            yml_content = f.read()
        # Room codes are 4 uppercase alphanumeric characters in the page content
        for candidate in re.findall(r'\b([A-Z0-9]{4})\b', yml_content):
            if not candidate.startswith("ref") and candidate not in ("CLIC", "TRAI", "HTTP", "ROOM", "CODE"):
                room_code = candidate
                break
    except FileNotFoundError:
        pass
print(f"  Room code: {room_code}")
if not room_code:
    print("  ERROR: Could not find room code")
    cleanup_sessions()
    sys.exit(1)

# ------------------------------------------------------------------
# Step 3: Player 2 joins and registers
print(f"\n--- Step 3: Player 2 joins room {room_code} ---")
pw("p2", "open", APP_URL)
time.sleep(1)
refs = snap("p2")
code_ref = find_ref(refs, "textbox", "ROOM") or find_ref(refs, "textbox")
if not code_ref:
    print("  ERROR: Could not find code input")
    cleanup_sessions()
    sys.exit(1)
pw("p2", "fill", code_ref, f'"{room_code}"')
time.sleep(0.5)

refs = snap("p2")
join_ref = find_ref(refs, "Join Room")
if not join_ref:
    print("  ERROR: Could not find Join Room button")
    cleanup_sessions()
    sys.exit(1)
pw("p2", "click", join_ref)
time.sleep(1)

print("  Player 2 registers as Bob")
refs = snap("p2")
name_ref = find_ref(refs, "textbox")
if not name_ref:
    print("  ERROR: Could not find name input for player 2")
    cleanup_sessions()
    sys.exit(1)
pw("p2", "fill", name_ref, '"Bob"')
time.sleep(0.5)

refs = snap("p2")
submit_ref = find_ref(refs, "button", "Join") or find_ref(refs, "button", "Enter") or find_ref(refs, "button", "Go")
if not submit_ref:
    for ref, desc in refs.items():
        if "button" in desc.lower() and "create" not in desc.lower():
            submit_ref = ref
            break
if not submit_ref:
    print("  ERROR: Could not find submit button for player 2")
    cleanup_sessions()
    sys.exit(1)
pw("p2", "click", submit_ref)
time.sleep(1)

# ------------------------------------------------------------------
# Step 4: Verify players table
print("\n--- Step 4: Verify players in DB ---")
player_count = int(sql("SELECT COUNT(*) FROM players") or "0")
assert_ge("players table rows", 2, player_count)

# ------------------------------------------------------------------
# Step 5: Both ready up
print("\n--- Step 5: Both players ready up ---")
refs = snap("p1")
ready_ref = find_ref(refs, "Ready") or find_ref(refs, "ready")
if ready_ref:
    pw("p1", "click", ready_ref)
    print("  Player 1 readied up")
else:
    print("  WARNING: Could not find ready button for p1")
time.sleep(0.5)

refs = snap("p2")
ready_ref = find_ref(refs, "Ready") or find_ref(refs, "ready")
if ready_ref:
    pw("p2", "click", ready_ref)
    print("  Player 2 readied up")
else:
    print("  WARNING: Could not find ready button for p2")

# Wait for countdown + round start + WS connect
print("  Waiting for countdown + game start...")
time.sleep(5)

# ------------------------------------------------------------------
# Step 6: Verify games table
print("\n--- Step 6: Verify game record in DB ---")
game_count = int(sql("SELECT COUNT(*) FROM games") or "0")
assert_ge("games table rows", 1, game_count)

# ------------------------------------------------------------------
# Step 7: Click targets via JS (SVG targets don't get accessibility refs)
# The game listens for mousedown on circle[data-points] elements
print("\n--- Step 7: Click targets ---")
click_js = "() => { const circles = document.querySelectorAll('circle[data-points]'); let clicked = 0; for (const c of circles) { if (clicked >= 2) break; c.dispatchEvent(new MouseEvent('mousedown', {bubbles: true})); clicked++; } return clicked; }"
for session, player in [("p1", "P1"), ("p2", "P2")]:
    for attempt in range(3):
        out = pw_eval(session, click_js)
        print(f"  {player} click attempt {attempt+1}: {out}")
        time.sleep(1)

# Wait for round to end + flush
print(f"  Waiting for round to end + DB flush...")
time.sleep(ROUND_DURATION + 5)

# ------------------------------------------------------------------
# Step 8: Verify all DB tables
print("\n--- Step 8: Verify DB after game ends ---")

game_count = int(sql("SELECT COUNT(*) FROM games") or "0")
assert_ge("games rows", 1, game_count)

ended_count = int(sql("SELECT COUNT(*) FROM games WHERE ended_at IS NOT NULL") or "0")
assert_ge("games with ended_at", 1, ended_count)

gp_count = int(sql("SELECT COUNT(*) FROM game_players") or "0")
assert_ge("game_players rows", 2, gp_count)

click_count = int(sql("SELECT COUNT(*) FROM click_events") or "0")
assert_ge("click_events rows", 1, click_count)

if click_count > 0:
    min_reaction = int(sql("SELECT MIN(reaction_ms) FROM click_events") or "0")
    assert_gt("min reaction_ms positive", 0, min_reaction)

# ------------------------------------------------------------------
# Step 9: Screenshot
print("\n--- Step 9: Final screenshots ---")
pw("p1", "screenshot", "--filename=screenshots/integration_p1_final.png")
pw("p2", "screenshot", "--filename=screenshots/integration_p2_final.png")

# Cleanup
cleanup_sessions()

print(f"\n=== Results: {passed} passed, {failed} failed ===")
sys.exit(0 if failed == 0 else 1)
