#!/usr/bin/env python3
"""Click Trainer Bot Swarm — spawn multiple bots into the same room.

Usage:
  python scripts/swarm.py [OPTIONS]

Options:
  --server URL        Base URL (default: http://localhost:8080)
  --room-code CODE    Join an existing room; omit to have the first bot create one
  --count N           Number of bots (default: 3)
  --skill PRESET      Skill preset for all bots: beginner|intermediate|expert|elite
  --skills S1,S2,...  Per-bot skill list (cycled if shorter than --count)
  --rounds N          Rounds to play then exit (default: 1)
  --stagger-ms N      Delay in ms between launching follower bots (default: 800)
  --verbose           Pass --verbose to every bot subprocess
"""

import argparse
import json
import os
import queue
import random
import string
import subprocess
import sys
import threading
import time

# ---------------------------------------------------------------------------
# Constants
# ---------------------------------------------------------------------------

SKILL_PRESETS = ["beginner", "intermediate", "expert", "elite"]

# ANSI colours — cycled per bot index
COLORS = ["\033[92m", "\033[94m", "\033[93m", "\033[95m", "\033[96m", "\033[91m"]
RESET  = "\033[0m"
BOLD   = "\033[1m"

BOT_SCRIPT = os.path.join(os.path.dirname(__file__), "bot.py")


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def color_for(index: int) -> str:
    return COLORS[index % len(COLORS)]


def random_suffix(n: int = 3) -> str:
    return "".join(random.choices(string.ascii_uppercase, k=n))


def build_cmd(server: str, room_code: str | None, name: str, skill: str,
              rounds: int, verbose: bool, count: int = 1,
              screenshots: bool = False,
              screenshot_prefix: str | None = None) -> list[str]:
    cmd = [
        sys.executable, BOT_SCRIPT,
        "--server", server,
        "--name", name,
        "--skill", skill,
        "--rounds", str(rounds),
        "--min-players", str(count),
    ]
    if room_code:
        cmd += ["--room-code", room_code]
    if verbose:
        cmd.append("--verbose")
    if screenshots:
        cmd.append("--screenshots")
    if screenshot_prefix:
        cmd += ["--screenshot-prefix", screenshot_prefix]
    return cmd


# ---------------------------------------------------------------------------
# Per-bot runner (runs in its own thread)
# ---------------------------------------------------------------------------

def run_bot(index: int, name: str, cmd: list[str],
            out_q: queue.Queue,
            room_code_ready: threading.Event,
            room_code_box: list,   # mutable single-element list
            verbose: bool):
    """Launch one bot subprocess, forward its JSON events to out_q."""

    # Followers wait until bot 0 has published the room code
    if index > 0:
        if not room_code_ready.wait(timeout=90):
            out_q.put({
                "_bot": name, "_index": index, "_type": "error",
                "message": "timed out waiting for room code from bot 0",
            })
            return

    # Inject the room code now that we know it
    final_cmd = cmd[:]
    if index > 0 and room_code_box[0] and "--room-code" not in final_cmd:
        final_cmd += ["--room-code", room_code_box[0]]

    proc = subprocess.Popen(final_cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)

    # Drain stderr in a background thread so it never blocks stdout reads
    def drain_stderr():
        for line in proc.stderr:
            if verbose:
                out_q.put({
                    "_bot": name, "_index": index, "_type": "stderr",
                    "line": line.rstrip(),
                })
    threading.Thread(target=drain_stderr, daemon=True).start()

    for raw_line in proc.stdout:
        raw_line = raw_line.strip()
        if not raw_line:
            continue
        try:
            event = json.loads(raw_line)
        except json.JSONDecodeError:
            event = {"_raw": raw_line}
        event["_bot"] = name
        event["_index"] = index
        out_q.put(event)

        # Bot 0 creating a new room: capture the code for all followers
        if index == 0 and event.get("event") == "registered" and "room_code" in event:
            if not room_code_box[0]:
                room_code_box[0] = event["room_code"]
            room_code_ready.set()

    proc.wait()
    out_q.put({
        "_bot": name, "_index": index, "_type": "exit",
        "returncode": proc.returncode,
    })


# ---------------------------------------------------------------------------
# Event display
# ---------------------------------------------------------------------------

def fmt_event(event: dict) -> str:
    """Return a human-readable one-liner for a bot event."""
    ev = event.get("event", "")
    if ev == "registered":
        return f"joined room {event.get('room_code')} as {event.get('name')}"
    if ev == "combat_start":
        return "combat started"
    if ev == "click":
        return (f"clicked target {event.get('target_id')} "
                f"+{event.get('points')}pt  "
                f"(total {event.get('cumulative_score')}pt, "
                f"react {event.get('reaction_ms')}ms)")
    if ev == "miss":
        return f"missed target {event.get('target_id')}  [{event.get('reason')}]"
    if ev == "recap":
        players = event.get("players") or []
        board = "  ".join(f"{p['name']}:{p['score']}pt" for p in players)
        return f"recap round {event.get('round')}  |  {board}"
    if ev == "error":
        return f"ERROR: {event.get('message')}"
    if ev == "interrupted":
        return "interrupted"
    # fallback: dump without internal keys
    display = {k: v for k, v in event.items() if not k.startswith("_")}
    return json.dumps(display)


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def parse_args():
    p = argparse.ArgumentParser(description="Click Trainer Bot Swarm")
    p.add_argument("--server", default="http://localhost:8080")
    p.add_argument("--room-code", default=None)
    p.add_argument("--count", type=int, default=3)
    p.add_argument("--skill", default=None, choices=SKILL_PRESETS)
    p.add_argument("--skills", default=None,
                   help="Comma-separated skill presets, e.g. beginner,expert,elite")
    p.add_argument("--rounds", type=int, default=1)
    p.add_argument("--stagger-ms", type=int, default=800,
                   help="Ms between launching each follower bot (default 800)")
    p.add_argument("--verbose", action="store_true")
    p.add_argument("--screenshots", action="store_true",
                   help="Capture desktop + portrait screenshots (passed to bot 0 only)")
    p.add_argument("--screenshot-prefix", default=None,
                   help="Filename prefix for screenshots (default: auto)")
    return p.parse_args()


def main():
    args = parse_args()

    # Resolve skill list
    if args.skills:
        skill_list = [s.strip() for s in args.skills.split(",")]
    elif args.skill:
        skill_list = [args.skill]
    else:
        # Mixed by default
        skill_list = SKILL_PRESETS
    skills = [skill_list[i % len(skill_list)] for i in range(args.count)]

    # Generate unique bot names
    names = [f"Bot{i+1}-{random_suffix()}" for i in range(args.count)]

    # Shared state
    out_q: queue.Queue = queue.Queue()
    room_code_ready = threading.Event()
    room_code_box: list = [args.room_code]  # [0] holds the code; None until bot 0 creates room
    if args.room_code:
        room_code_ready.set()  # already known — all bots can launch immediately

    print(f"{BOLD}Swarm: {args.count} bots  skill={skills}  rounds={args.rounds}{RESET}")
    if args.room_code:
        print(f"  joining existing room {args.room_code.upper()}")
    else:
        print("  bot 0 will create a new room")
    print()

    # Launch threads — bot 0 first, followers staggered
    threads: list[threading.Thread] = []
    for i, (name, skill) in enumerate(zip(names, skills)):
        # Only bot 0 takes screenshots (it creates the room and sees all phases)
        take_shots = args.screenshots and i == 0
        cmd = build_cmd(args.server, args.room_code, name, skill, args.rounds,
                        args.verbose, count=args.count, screenshots=take_shots,
                        screenshot_prefix=args.screenshot_prefix)
        t = threading.Thread(
            target=run_bot,
            args=(i, name, cmd, out_q, room_code_ready, room_code_box, args.verbose),
            daemon=True,
        )
        t.start()
        threads.append(t)

        # Stagger followers so they don't all hammer the page at once
        if i < args.count - 1:
            time.sleep(args.stagger_ms / 1000.0)

    # Collect per-bot final scores for summary
    scores: dict[str, int] = {}   # name → last cumulative_score seen
    active = args.count

    try:
        while active > 0:
            try:
                event = out_q.get(timeout=1.0)
            except queue.Empty:
                continue

            idx  = event.get("_index", 0)
            name = event.get("_bot", "?")
            col  = color_for(idx)
            tag  = f"{col}[{name}]{RESET}"

            etype = event.get("_type")

            if etype == "exit":
                active -= 1
                rc = event.get("returncode", 0)
                status = "done" if rc == 0 else f"exited ({rc})"
                print(f"{tag} {status}")
                continue

            if etype == "stderr":
                print(f"{tag} {event['line']}", file=sys.stderr)
                continue

            if etype == "error":
                print(f"{tag} {BOLD}ERROR{RESET}: {event['message']}")
                active -= 1
                continue

            # Track score for summary
            if event.get("event") == "click":
                scores[name] = event.get("cumulative_score", 0)

            print(f"{tag} {fmt_event(event)}")

    except KeyboardInterrupt:
        print(f"\n{BOLD}Interrupted — bots will clean up their browser sessions.{RESET}")

    # Wait for threads to finish (they're daemons so the process won't hang)
    for t in threads:
        t.join(timeout=5)

    # Final summary
    if scores:
        print(f"\n{BOLD}--- Final Scores ---{RESET}")
        ranked = sorted(scores.items(), key=lambda x: x[1], reverse=True)
        for rank, (name, score) in enumerate(ranked, 1):
            idx = names.index(name) if name in names else 0
            col = color_for(idx)
            skill = skills[names.index(name)] if name in names else "?"
            print(f"  {rank}. {col}{name}{RESET}  {score}pt  [{skill}]")


if __name__ == "__main__":
    main()
