#!/usr/bin/env python3
"""Edge case tests for Click Trainer.

Covers three scenarios:
  E1 — Invalid room code        → "Room not found" error on home page
  E2 — Last player leaves       → room deleted; old code redirects to /
  E3 — Deep link, no session    → join/register form rendered (not a redirect)

Usage:
  python3 scripts/test_edge.py [--server URL]

Prerequisites:
  - Server running at localhost:8080 (or --server URL)
  - playwright-cli available on PATH
  - Run from the repo root (screenshots/ is created there)

Exit code: 0 if all checks pass, 1 if any fail.
"""

import argparse
import os
import re
import subprocess
import sys
import time
import urllib.request


# ---------------------------------------------------------------------------
# Global state
# ---------------------------------------------------------------------------

APP_URL = "http://localhost:8080"
_passed = 0
_failed = 0


# ---------------------------------------------------------------------------
# Playwright helpers
# ---------------------------------------------------------------------------

def _pw_run(session: str, args: list[str], timeout: int = 15) -> str:
    cmd = ["playwright-cli", f"-s={session}"] + args
    result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
    return result.stdout + result.stderr


def pw(session: str, *args: str) -> str:
    return _pw_run(session, list(args))


def pw_eval(session: str, js: str) -> str:
    """Run JS via playwright-cli eval; return the unwrapped result string."""
    out = _pw_run(session, ["eval", js])
    for line in out.splitlines():
        line = line.strip()
        if line and not line.startswith(("#", "```", "await", "//", "###")):
            # playwright-cli wraps string results in double-quotes; strip them.
            if line.startswith('"') and line.endswith('"') and len(line) >= 2:
                line = line[1:-1].replace('\\"', '"')
            return line
    return out.strip()


def snap(session: str) -> dict[str, str]:
    """Take a snapshot; return {ref_id: description_line} dict."""
    out = pw(session, "snapshot")
    m = re.search(r'\((.+?\.yml)\)', out)
    if not m:
        return {}
    yml_path = os.path.join(os.getcwd(), m.group(1))
    try:
        with open(yml_path) as f:
            content = f.read()
    except FileNotFoundError:
        return {}
    refs = {}
    for line in content.splitlines():
        hit = re.search(r'\[ref=(\w+)\]', line)
        if hit:
            refs[hit.group(1)] = line.strip()
    return refs


def find_ref(refs: dict[str, str], *keywords: str) -> str | None:
    """Return the first ref whose description contains all keywords (case-insensitive)."""
    for ref, desc in refs.items():
        lower = desc.lower()
        if all(k.lower() in lower for k in keywords):
            return ref
    return None


def screenshot(session: str, filename: str) -> str:
    """Save a screenshot to screenshots/{filename}. Returns the path."""
    path = f"screenshots/{filename}"
    pw(session, "screenshot", f"--filename={path}")
    return path


def current_url(session: str) -> str:
    return pw_eval(session, "window.location.href")


def close(session: str):
    try:
        pw(session, "close")
    except Exception:
        pass


# ---------------------------------------------------------------------------
# Assert helpers
# ---------------------------------------------------------------------------

def check(label: str, condition: bool, detail: str = "") -> bool:
    global _passed, _failed
    if condition:
        print(f"  PASS: {label}")
        _passed += 1
    else:
        print(f"  FAIL: {label}")
        if detail:
            print(f"        → {detail}")
        _failed += 1
    return condition


# ---------------------------------------------------------------------------
# E1 — Invalid room code
# ---------------------------------------------------------------------------

def test_e1_invalid_room_code():
    """Submitting a non-existent 4-char code re-renders home with an error."""
    print("\n--- E1: Invalid room code (ZZZZ) ---")
    session = "edge_e1"
    try:
        pw(session, "open", APP_URL)
        time.sleep(0.5)

        refs = snap(session)
        code_ref = find_ref(refs, "textbox", "ROOM") or find_ref(refs, "textbox")
        if not code_ref:
            check("E1.1: Room code input visible on home page", False,
                  "Could not find a textbox ref in the home page snapshot")
            return

        pw(session, "fill", code_ref, "ZZZZ")
        time.sleep(0.3)

        refs = snap(session)
        join_ref = find_ref(refs, "Join Room")
        if not join_ref:
            check("E1.1: Join Room button visible", False,
                  "Could not find 'Join Room' button")
            return

        pw(session, "click", join_ref)
        time.sleep(0.5)

        screenshot(session, "edge_e1_invalid_code.png")

        # E1.1 — URL stays away from /room/ path
        url = current_url(session)
        check("E1.1: URL does not contain /room/ after bad join",
              "/room/" not in url,
              f"actual URL: {url}")

        # E1.2 — Error message element present
        error_text = pw_eval(
            session,
            "document.querySelector('.error-msg')?.textContent ?? ''"
        )
        check("E1.2: .error-msg element exists on page",
              bool(error_text),
              "No element with class 'error-msg' found in DOM")

        # E1.3 — Error message contains expected text
        check("E1.3: .error-msg text is 'Room not found'",
              "Room not found" in error_text,
              f"actual text: {repr(error_text)}")

    finally:
        close(session)


# ---------------------------------------------------------------------------
# E2 — Last player leaves → room deleted
# ---------------------------------------------------------------------------

def test_e2_last_player_leaves():
    """The last player leaving a room deletes it; the old code then redirects to /."""
    print("\n--- E2: Last player leaves → room deleted ---")
    creator = "edge_e2_creator"
    visitor = "edge_e2_visitor"
    room_code = None

    try:
        # — Create a room —
        pw(creator, "open", APP_URL)
        time.sleep(0.5)

        refs = snap(creator)
        create_ref = find_ref(refs, "Create Room")
        if not create_ref:
            check("E2.setup: Create Room button visible", False,
                  "Could not find 'Create Room' button on home page")
            return

        pw(creator, "click", create_ref)
        time.sleep(0.8)

        url = current_url(creator)
        m = re.search(r'/room/([A-Z0-9]{4})', url)
        if not m:
            check("E2.setup: Room URL contains 4-char code", False,
                  f"No room code found in URL: {url}")
            return
        room_code = m.group(1)
        print(f"  Created room: {room_code}")

        # — Register as 'Leaver' —
        refs = snap(creator)
        name_ref = find_ref(refs, "textbox")
        if not name_ref:
            check("E2.setup: Name input on register page", False,
                  "Could not find name textbox on register page")
            return

        pw(creator, "fill", name_ref, "Leaver")
        time.sleep(0.3)

        refs = snap(creator)
        submit_ref = (
            find_ref(refs, "button", "Start") or
            find_ref(refs, "button", "Join") or
            find_ref(refs, "button", "Enter") or
            find_ref(refs, "button", "Go")
        )
        if not submit_ref:
            check("E2.setup: Submit button on register page", False,
                  "Could not find submit button on register page")
            return

        pw(creator, "click", submit_ref)
        time.sleep(1.0)

        # — Leave the room —
        refs = snap(creator)
        leave_ref = find_ref(refs, "Leave Room") or find_ref(refs, "Leave")
        if not leave_ref:
            check("E2.setup: Leave Room button visible in lobby", False,
                  "Could not find Leave Room button in lobby snapshot")
            return

        pw(creator, "click", leave_ref)
        time.sleep(0.8)

        screenshot(creator, "edge_e2_after_leave.png")

        # E2.1 — Creator lands back on home
        creator_url = current_url(creator)
        check("E2.1: Leaving redirects creator away from /room/",
              "/room/" not in creator_url,
              f"actual URL: {creator_url}")

        # — Visitor navigates to the now-deleted room code —
        time.sleep(0.3)
        pw(visitor, "open", f"{APP_URL}/room/{room_code}")
        time.sleep(0.5)

        screenshot(visitor, "edge_e2_deleted_room.png")

        visitor_url = current_url(visitor)
        check("E2.2: Accessing deleted room redirects visitor away from /room/",
              "/room/" not in visitor_url,
              f"actual URL: {visitor_url}")

    finally:
        close(creator)
        close(visitor)


# ---------------------------------------------------------------------------
# E3 — Deep link without session cookie
# ---------------------------------------------------------------------------

def test_e3_deep_link_no_session():
    """Navigating directly to /room/{code} without a session renders the
    join/register form — the server does not redirect to /."""
    print("\n--- E3: Deep link without session cookie ---")
    host = "edge_e3_host"
    visitor = "edge_e3_visitor"

    try:
        # — Create an active room so there is a valid code to deep-link to —
        pw(host, "open", APP_URL)
        time.sleep(0.5)

        refs = snap(host)
        create_ref = find_ref(refs, "Create Room")
        if not create_ref:
            check("E3.setup: Create Room button visible", False,
                  "Could not find 'Create Room' button on home page")
            return

        pw(host, "click", create_ref)
        time.sleep(0.8)

        url = current_url(host)
        m = re.search(r'/room/([A-Z0-9]{4})', url)
        if not m:
            check("E3.setup: Room URL contains 4-char code", False,
                  f"No room code in URL: {url}")
            return
        room_code = m.group(1)
        print(f"  Active room: {room_code}")

        # Register the host player so the room persists in memory
        refs = snap(host)
        name_ref = find_ref(refs, "textbox")
        if name_ref:
            pw(host, "fill", name_ref, "Host")
            time.sleep(0.3)
            refs = snap(host)
            submit_ref = (
                find_ref(refs, "button", "Start") or
                find_ref(refs, "button", "Join") or
                find_ref(refs, "button", "Enter")
            )
            if submit_ref:
                pw(host, "click", submit_ref)
                time.sleep(0.8)

        # — Fresh session deep-links directly to the room (no session cookie) —
        pw(visitor, "open", f"{APP_URL}/room/{room_code}")
        time.sleep(0.5)

        screenshot(visitor, "edge_e3_deep_link.png")

        visitor_url = current_url(visitor)

        # E3.1 — Not redirected away; stays on /room/{code}
        check("E3.1: Deep link keeps URL on /room/{code} (no redirect to /)",
              f"/room/{room_code}" in visitor_url,
              f"actual URL: {visitor_url}")

        # E3.2 — Page title signals the join template
        title = pw_eval(visitor, "document.title")
        check("E3.2: Page title indicates join/register template",
              "Join" in title or "join" in title.lower(),
              f"actual title: {repr(title)}")

        # E3.3 — A name input field is rendered
        has_input = pw_eval(
            visitor,
            "document.querySelector('input') !== null"
        )
        check("E3.3: Name input field is present (register form rendered)",
              has_input == "true",
              f"querySelector('input') returned: {repr(has_input)}")

        # E3.4 — Lobby element is absent (player hasn't registered yet)
        has_lobby = pw_eval(
            visitor,
            "document.querySelector('#lobby') !== null"
        )
        check("E3.4: #lobby element is NOT present (player not yet registered)",
              has_lobby == "false",
              f"querySelector('#lobby') returned: {repr(has_lobby)}")

    finally:
        close(host)
        close(visitor)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Edge case tests for Click Trainer")
    p.add_argument("--server", default="http://localhost:8080",
                   help="Base URL of the app (default: http://localhost:8080)")
    return p.parse_args()


def main():
    global APP_URL
    args = parse_args()
    APP_URL = args.server.rstrip("/")

    os.makedirs("screenshots", exist_ok=True)

    print("=== Click Trainer — Edge Case Tests ===")

    # Health check
    try:
        with urllib.request.urlopen(f"{APP_URL}/health", timeout=5) as resp:
            body = resp.read().decode()
        if '"status":"ok"' not in body:
            print(f"ERROR: Unexpected health response: {body}")
            sys.exit(1)
        print(f"Server at {APP_URL} is healthy.\n")
    except Exception as exc:
        print(f"ERROR: Cannot reach {APP_URL}/health — {exc}")
        sys.exit(1)

    test_e1_invalid_room_code()
    test_e2_last_player_leaves()
    test_e3_deep_link_no_session()

    print(f"\n=== Results: {_passed} passed, {_failed} failed ===")
    sys.exit(0 if _failed == 0 else 1)


if __name__ == "__main__":
    main()
