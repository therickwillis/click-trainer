#!/usr/bin/env python3
"""Generate screenshots of Click Trainer by running bots through a full game.

Usage:
  python3 generate-screenshots.py [--players N] [--rounds N] [--prefix PREFIX]

Examples:
  python3 generate-screenshots.py
  python3 generate-screenshots.py --players 3
  python3 generate-screenshots.py --players 2 --rounds 2 --prefix v2
"""

import argparse
import os
import subprocess
import sys

SCRIPTS_DIR = os.path.join(os.path.dirname(__file__), "")
BOT_SCRIPT   = os.path.join(SCRIPTS_DIR, "bot.py")
SWARM_SCRIPT = os.path.join(SCRIPTS_DIR, "swarm.py")


def parse_args():
    p = argparse.ArgumentParser(description="Generate Click Trainer screenshots")
    p.add_argument("--players", type=int, default=1,
                   help="Number of bot players (default: 1)")
    p.add_argument("--rounds", type=int, default=1,
                   help="Rounds to play (default: 1)")
    p.add_argument("--prefix", default="screenshots",
                   help="Screenshot filename prefix (default: screenshots)")
    p.add_argument("--server", default="http://localhost:8080",
                   help="Server URL (default: http://localhost:8080)")
    p.add_argument("--skill", default="elite",
                   choices=["beginner", "intermediate", "expert", "elite"],
                   help="Bot skill level (default: elite)")
    return p.parse_args()


def main():
    args = parse_args()

    if args.players < 1:
        print("--players must be at least 1", file=sys.stderr)
        sys.exit(1)

    print(f"Players: {args.players}  Rounds: {args.rounds}  "
          f"Skill: {args.skill}  Prefix: screenshots/{args.prefix}_*")
    print()

    if args.players == 1:
        cmd = [
            sys.executable, BOT_SCRIPT,
            "--server", args.server,
            "--name", "Solo",
            "--skill", args.skill,
            "--rounds", str(args.rounds),
            "--screenshots",
            "--screenshot-prefix", args.prefix,
        ]
    else:
        cmd = [
            sys.executable, SWARM_SCRIPT,
            "--server", args.server,
            "--count", str(args.players),
            "--skill", args.skill,
            "--rounds", str(args.rounds),
            "--stagger-ms", "1000",
            "--screenshots",
            "--screenshot-prefix", args.prefix,
        ]

    result = subprocess.run(cmd)
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
