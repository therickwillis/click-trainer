#!/usr/bin/env python3
"""Click Trainer Bot — simulates a human player via playwright-cli.

Usage:
  python scripts/bot.py [OPTIONS]

Options:
  --server URL        Base URL (default: http://localhost:8080)
  --room-code CODE    Join existing room (4-char code); omit to create new room
  --name NAME         Bot player name (default: "Bot-{random}")
  --skill PRESET      Skill preset: beginner|intermediate|expert|elite (default: intermediate)
  --reaction-ms N     Override base reaction delay in ms
  --miss-rate F       Override miss rate (0.0–1.0)
  --accuracy F        Override accuracy: 0.0=always outer (1pt), 1.0=always center (4pt)
  --max-cps F         Override max clicks per second
  --rounds N          Play N rounds then exit (default: 1)
  --verbose           Print debug info (snapshot dumps, timing)
  --screenshots       Capture desktop + portrait screenshots at key phases
  --screenshot-prefix PREFIX
                      Filename prefix (default: derived from bot name + timestamp)
"""

import argparse
import json
import os
import random
import re
import string
import subprocess
import sys
import time
from dataclasses import dataclass, field
from typing import Optional


# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

SKILL_PRESETS = {
    "beginner":     dict(reaction_ms=1200, jitter=0.30, accuracy=0.0,  miss_rate=0.30, max_cps=0.8),
    "intermediate": dict(reaction_ms=600,  jitter=0.20, accuracy=0.5,  miss_rate=0.15, max_cps=1.5),
    "expert":       dict(reaction_ms=300,  jitter=0.15, accuracy=0.75, miss_rate=0.08, max_cps=2.5),
    "elite":        dict(reaction_ms=100,  jitter=0.10, accuracy=0.9,  miss_rate=0.02, max_cps=5.0),
}


@dataclass
class BotConfig:
    server: str = "http://localhost:8080"
    room_code: Optional[str] = None
    name: str = ""
    skill: str = "intermediate"
    reaction_ms: float = 600.0
    jitter: float = 0.20
    accuracy: float = 0.5
    miss_rate: float = 0.15
    max_cps: float = 1.5
    rounds: int = 1
    verbose: bool = False
    session: str = ""
    screenshots: bool = False
    screenshot_prefix: str = ""


# ---------------------------------------------------------------------------
# Playwright CLI wrapper
# ---------------------------------------------------------------------------

class PlaywrightSession:
    def __init__(self, session: str, verbose: bool = False):
        self.session = session
        self.verbose = verbose

    def run(self, *args: str, timeout: int = 15) -> str:
        cmd = ["playwright-cli", f"-s={self.session}"] + list(args)
        if self.verbose:
            print(f"[pw] {' '.join(cmd)}", file=sys.stderr)
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=timeout)
        out = result.stdout + result.stderr
        if self.verbose and out.strip():
            preview = out[:500] + ("..." if len(out) > 500 else "")
            print(f"[pw out] {preview}", file=sys.stderr)
        return out

    def open(self, url: str) -> str:
        return self.run("open", url)

    def click(self, ref: str) -> str:
        return self.run("click", ref)

    def fill(self, ref: str, value: str) -> str:
        return self.run("fill", ref, value)

    def close(self) -> str:
        return self.run("close")

    def screenshot(self, name: str) -> str:
        """Take screenshot into screenshots/{name}. Returns the full path."""
        path = f"screenshots/{name}"
        self.run("screenshot", f"--filename={path}", timeout=10)
        return path

    def eval(self, js: str) -> str:
        """Run JS via playwright-cli eval and return the result string.

        playwright-cli wraps string results in double quotes, e.g. "combat" or "[1,2]".
        This method strips those outer quotes so callers get the raw value.
        """
        result = subprocess.run(
            ["playwright-cli", f"-s={self.session}", "eval", js],
            capture_output=True, text=True, timeout=15,
        )
        out = result.stdout + result.stderr
        if self.verbose:
            print(f"[eval] {js[:80]}... → {out[:100]}", file=sys.stderr)
        # Extract the result line (skip comment/fence lines)
        for line in out.splitlines():
            line = line.strip()
            if line and not line.startswith("#") and not line.startswith("```") and not line.startswith("await"):
                # Strip the outer double-quotes playwright-cli adds around string results
                if line.startswith('"') and line.endswith('"') and len(line) >= 2:
                    line = line[1:-1]
                    # Unescape internal escaped quotes
                    line = line.replace('\\"', '"')
                return line
        return out.strip()

    def snapshot_refs(self) -> dict[str, str]:
        """Take snapshot; return {ref: line} dict from the YAML file."""
        out = self.run("snapshot")
        m = re.search(r'\((.+?\.yml)\)', out)
        if not m:
            return {}
        yml_path = os.path.join("/workspace", m.group(1))
        try:
            with open(yml_path) as f:
                content = f.read()
            return _parse_refs(content)
        except FileNotFoundError:
            return {}


def _parse_refs(text: str) -> dict[str, str]:
    refs = {}
    for line in text.splitlines():
        m = re.search(r'\[ref=(\w+)\]', line)
        if m:
            refs[m.group(1)] = line.strip()
    return refs


def find_ref(refs: dict[str, str], *keywords: str) -> Optional[str]:
    for ref, desc in refs.items():
        lower = desc.lower()
        if all(k.lower() in lower for k in keywords):
            return ref
    return None


# ---------------------------------------------------------------------------
# Output helpers
# ---------------------------------------------------------------------------

def emit(event: dict):
    print(json.dumps(event), flush=True)


# ---------------------------------------------------------------------------
# JS helpers (baked constants avoid any quoting/injection concerns)
# ---------------------------------------------------------------------------

JS_GET_SCENE = "() => document.body.getAttribute('data-scene')"

JS_GET_TARGETS = (
    "() => JSON.stringify("
    "  Array.from(document.querySelectorAll('[data-target-id]'))"
    "  .map(d => parseInt(d.getAttribute('data-target-id')))"
    ")"
)

JS_GET_PLAYER_ID = (
    "() => {"
    # scoreboard element (combat scene): id="player_score_<uuid>"
    "  var el = document.querySelector('[id^=\"player_score_\"]');"
    "  if (el) return el.id.replace('player_score_', '');"
    # lobby player elements (lobby scene): id="lobby_player<uuid>"
    # Note: the container is id="lobby_players" — filter it out by requiring length > 10
    "  var els = document.querySelectorAll('[id^=\"lobby_player\"]');"
    "  for (var i = 0; i < els.length; i++) {"
    "    var id = els[i].id.replace('lobby_player', '');"
    "    if (id.length > 10) return id;"
    "  }"
    "  return '';"
    "}"
)

JS_GET_RECAP = (
    "() => JSON.stringify("
    "  Array.from(document.querySelectorAll('#recap .flex-row')).map(row => {"
    "    var cells = row.querySelectorAll('div');"
    "    var name = '', score = 0;"
    "    cells.forEach(c => {"
    "      var t = c.textContent.trim();"
    "      if (t.endsWith(' pts')) score = parseInt(t);"
    "      else if (t.length > 1 && !t.match(/^\\d+$/)) name = t;"
    "    });"
    "    return {name: name, score: score};"
    "  }).filter(p => p.name)"
    ")"
)

# Simulate a 393×852 portrait viewport on the existing 1280×720 browser window.
# Directly applies mobile CSS rules (skipped because media query won't fire at 1280px),
# forces the game rotation, and adds an orange frame label so the sim is obvious.
JS_INJECT_PORTRAIT_SIM = (
    "() => {"
    # Portrait dimensions and derived scale (400-wide × 600-tall after -90° rotate)
    "  var PW=393, PH=852, HUD_H=46;"
    "  var scale = Math.min(PW*0.97/400, (PH-HUD_H)*0.97/600);"
    "  var root = document.documentElement;"
    "  root.style.setProperty('--game-scale', scale);"
    "  root.style.setProperty('--game-rotate', 1);"
    # CRITICAL: .game-scene has transform:scale(...) from CSS which (a) double-scales
    # the game-area and (b) traps position:fixed children, preventing HUD from escaping
    # to the viewport top. Override with 'none' to fix both issues.
    "  var scene = document.querySelector('.game-scene');"
    "  if (scene) scene.style.transform = 'none';"
    # Apply mobile #game-area styles directly (media query doesn't fire at 1280px)
    "  var ga = document.getElementById('game-area');"
    "  if (ga) {"
    "    ga.style.position='fixed';"
    "    ga.style.inset='0';"
    "    ga.style.margin='auto';"
    "    ga.style.width='600px';"
    "    ga.style.height='400px';"
    "    ga.style.transformOrigin='center center';"
    "    ga.style.transform='rotate(-90deg) scale('+scale+')';"
    "  }"
    # Apply mobile HUD styles
    "  var hud = document.querySelector('.game-hud');"
    "  if (hud) {"
    "    hud.style.position='fixed';"
    "    hud.style.top='0';"
    "    hud.style.left='0';"
    "    hud.style.right='0';"
    "    hud.style.zIndex='100';"
    "    hud.style.backdropFilter='blur(10px)';"
    "    hud.style.background='rgba(0,0,0,0.6)';"
    "    hud.style.padding='0.4rem 0.75rem';"
    "    hud.style.width='auto';"
    "  }"
    # Hide chip names (mobile behavior)
    "  document.querySelectorAll('.player-chip__name').forEach(function(el){"
    "    el.setAttribute('data-portrait-hidden','1');"
    "    el.style.display='none';"
    "  });"
    # Phone-frame overlay with label
    "  if (!document.getElementById('__portrait_frame__')) {"
    "    var frame = document.createElement('div');"
    "    frame.id = '__portrait_frame__';"
    "    frame.style.cssText = 'position:fixed;inset:0;pointer-events:none;z-index:9999;"
    "      box-shadow:inset 0 0 0 5px rgba(255,120,0,0.85);border-radius:16px';"
    "    var label = document.createElement('div');"
    "    label.style.cssText = 'position:absolute;bottom:8px;right:10px;"
    "      font:700 11px monospace;color:rgba(255,150,0,1);"
    "      background:rgba(0,0,0,0.7);padding:2px 8px;border-radius:4px;"
    "      letter-spacing:0.05em';"
    "    label.textContent = 'portrait sim 393\u00d7852';"
    "    frame.appendChild(label);"
    "    document.body.appendChild(frame);"
    "  }"
    "}"
)

# Undo the portrait simulation and restore desktop layout.
JS_RESTORE_DESKTOP = (
    "() => {"
    "  var vw=window.innerWidth, vh=window.innerHeight;"
    "  var scale = Math.min(vw*0.97/600, vh*0.97/446);"
    "  var root = document.documentElement;"
    "  root.style.setProperty('--game-scale', scale);"
    "  root.style.setProperty('--game-rotate', 0);"
    # Restore game-scene transform (CSS will re-apply scale via --game-scale)
    "  var scene = document.querySelector('.game-scene');"
    "  if (scene) scene.style.transform = '';"
    "  var ga = document.getElementById('game-area');"
    "  if (ga) {"
    "    ga.style.position='';"
    "    ga.style.inset='';"
    "    ga.style.margin='';"
    "    ga.style.width='';"
    "    ga.style.height='';"
    "    ga.style.transformOrigin='';"
    "    ga.style.transform='';"
    "  }"
    "  var hud = document.querySelector('.game-hud');"
    "  if (hud) {"
    "    hud.style.position='';"
    "    hud.style.top='';"
    "    hud.style.left='';"
    "    hud.style.right='';"
    "    hud.style.zIndex='';"
    "    hud.style.backdropFilter='';"
    "    hud.style.background='';"
    "    hud.style.padding='';"
    "    hud.style.width='';"
    "  }"
    "  document.querySelectorAll('[data-portrait-hidden]').forEach(function(el){"
    "    el.style.display='';"
    "    el.removeAttribute('data-portrait-hidden');"
    "  });"
    "  var frame=document.getElementById('__portrait_frame__');"
    "  if (frame) frame.remove();"
    "}"
)


def js_click_target(target_id: int, points: int) -> str:
    """Build JS that dispatches mousedown on the right circle of a target."""
    return (
        f"() => {{"
        f"  var d = document.querySelector('[data-target-id=\"{target_id}\"]');"
        f"  if (!d) return 'notfound';"
        f"  var c = d.querySelector('circle[data-points=\"{points}\"]');"
        f"  if (!c) c = d.querySelector('circle[data-points]');"  # fallback to any circle
        f"  if (!c) return 'nocircle';"
        f"  c.dispatchEvent(new MouseEvent('mousedown', {{bubbles: true, cancelable: true}}));"
        f"  return 'clicked:' + c.getAttribute('data-points');"
        f"}}"
    )


# ---------------------------------------------------------------------------
# Bot state machine
# ---------------------------------------------------------------------------

class BotPlayer:
    def __init__(self, config: BotConfig):
        self.cfg = config
        self.pw = PlaywrightSession(config.session, verbose=config.verbose)
        self.player_id: Optional[str] = None
        self.room_code: Optional[str] = None
        self.score: int = 0
        self._last_click_time: float = 0.0
        self._clicked_targets: set[int] = set()
        self._screenshots_taken: list[str] = []

    # ------------------------------------------------------------------
    # Screenshot helpers
    # ------------------------------------------------------------------

    def _take_phase_screenshots(self, phase: str, wait_s: float = 0.0):
        """Capture desktop + portrait-sim screenshots for a named game phase.

        Files land in screenshots/ as:
          {prefix}_{phase}_desktop.png
          {prefix}_{phase}_portrait_sim.png
        """
        if not self.cfg.screenshots:
            return

        if wait_s > 0:
            time.sleep(wait_s)

        prefix = self.cfg.screenshot_prefix

        # --- Desktop (1280×720 landscape) ---
        desktop_name = f"{prefix}_{phase}_desktop.png"
        desktop_path = self.pw.screenshot(desktop_name)
        self._screenshots_taken.append(desktop_path)
        emit({
            "event": "screenshot",
            "phase": phase,
            "viewport": "desktop_1280x720",
            "file": desktop_path,
        })

        # --- Portrait simulation (393×852 mobile) ---
        self.pw.eval(JS_INJECT_PORTRAIT_SIM)
        time.sleep(0.25)  # allow repaint after CSS injection
        portrait_name = f"{prefix}_{phase}_portrait_sim.png"
        portrait_path = self.pw.screenshot(portrait_name)
        self._screenshots_taken.append(portrait_path)
        emit({
            "event": "screenshot",
            "phase": phase,
            "viewport": "portrait_sim_393x852",
            "file": portrait_path,
        })

        # Restore desktop layout for continued interaction
        self.pw.eval(JS_RESTORE_DESKTOP)

    def print_screenshot_summary(self):
        """Print a human-readable list of all screenshots taken."""
        if not self._screenshots_taken:
            return
        print("\n--- Screenshots ---", file=sys.stderr)
        for path in self._screenshots_taken:
            print(f"  {path}", file=sys.stderr)
        print(f"  ({len(self._screenshots_taken)} files in screenshots/)", file=sys.stderr)

    # ------------------------------------------------------------------
    # Phase: navigate
    # ------------------------------------------------------------------

    def navigate(self):
        self.pw.open(self.cfg.server)
        time.sleep(1)

    # ------------------------------------------------------------------
    # Phase: create or join room
    # ------------------------------------------------------------------

    def create_or_join(self) -> str:
        if self.cfg.room_code:
            return self._join_existing(self.cfg.room_code)
        return self._create_room()

    def _create_room(self) -> str:
        refs = self.pw.snapshot_refs()
        create_ref = find_ref(refs, "Create Room")
        if not create_ref:
            raise RuntimeError(f"Cannot find 'Create Room' button. refs={refs}")
        self.pw.click(create_ref)
        time.sleep(1)
        return self._extract_room_code_from_url()

    def _join_existing(self, code: str) -> str:
        refs = self.pw.snapshot_refs()
        code_ref = find_ref(refs, "textbox", "ROOM") or find_ref(refs, "textbox")
        if not code_ref:
            raise RuntimeError("Cannot find room code input")
        self.pw.fill(code_ref, code.upper())
        time.sleep(0.3)
        refs = self.pw.snapshot_refs()
        join_ref = find_ref(refs, "Join Room")
        if not join_ref:
            raise RuntimeError("Cannot find 'Join Room' button")
        self.pw.click(join_ref)
        time.sleep(1)
        return code.upper()

    def _extract_room_code_from_url(self) -> str:
        """Extract room code from the current page URL via JS."""
        url = self.pw.eval("() => window.location.href")
        # URL is like http://localhost:8080/room/X2J7
        m = re.search(r'/room/([A-Z0-9]{4})', url)
        if m:
            return m.group(1)
        # Fallback: read from lobby page text
        out = self.pw.eval(
            "() => {"
            "  var spans = document.querySelectorAll('span');"
            "  for (var s of spans) {"
            "    var t = s.textContent.trim();"
            "    if (/^[A-Z0-9]{4}$/.test(t)) return t;"
            "  }"
            "  return '';"
            "}"
        )
        if out and re.match(r'^[A-Z0-9]{4}$', out.strip()):
            return out.strip()
        raise RuntimeError(f"Cannot extract room code from URL: {url}")

    # ------------------------------------------------------------------
    # Phase: register
    # ------------------------------------------------------------------

    def register(self):
        refs = self.pw.snapshot_refs()
        name_ref = find_ref(refs, "textbox")
        if not name_ref:
            raise RuntimeError("Cannot find name input on join page")
        self.pw.fill(name_ref, self.cfg.name)
        time.sleep(0.3)

        refs = self.pw.snapshot_refs()
        submit_ref = (
            find_ref(refs, "button", "Start")
            or find_ref(refs, "button", "Join")
            or find_ref(refs, "button", "Enter")
            or find_ref(refs, "button", "Go")
        )
        if not submit_ref:
            skip_words = {"create room", "join room", "leave", "share", "copy"}
            for ref, desc in refs.items():
                if "button" in desc.lower() and not any(w in desc.lower() for w in skip_words):
                    submit_ref = ref
                    break
        if not submit_ref:
            raise RuntimeError(f"Cannot find submit button on register page. refs={refs}")
        self.pw.click(submit_ref)
        time.sleep(1.5)

        # Extract player_id via JS — HTML element IDs are not in accessibility snapshot
        self._extract_player_id()

    def _extract_player_id(self):
        # The scoreboard renders via SSE after registration; retry a few times
        for _ in range(10):
            result = self.pw.eval(JS_GET_PLAYER_ID).strip()
            if result and result not in ("null", "undefined", "''", '""', ""):
                self.player_id = result.strip("'\"")
                return
            time.sleep(0.5)

    # ------------------------------------------------------------------
    # Phase: lobby ready
    # ------------------------------------------------------------------

    def lobby_ready(self):
        # Screenshot the lobby before readying up
        self._take_phase_screenshots("lobby")

        for _ in range(30):
            refs = self.pw.snapshot_refs()
            ready_ref = find_ref(refs, "Ready") or find_ref(refs, "ready")
            if ready_ref:
                self.pw.click(ready_ref)
                return
            time.sleep(0.5)
        raise RuntimeError("Timed out waiting for ready button in lobby")

    # ------------------------------------------------------------------
    # Phase: wait for combat to start
    # ------------------------------------------------------------------

    def wait_for_combat(self):
        """Poll until data-scene == 'combat'."""
        for _ in range(120):  # up to 60s
            scene = self.pw.eval(JS_GET_SCENE).strip()
            if self.cfg.verbose:
                print(f"[scene] {scene}", file=sys.stderr)
            if scene == "combat":
                return
            time.sleep(0.5)
        raise RuntimeError("Timed out waiting for combat to start")

    # ------------------------------------------------------------------
    # Phase: combat
    # ------------------------------------------------------------------

    def combat(self):
        emit({"event": "combat_start", "ts": int(time.time())})

        # Screenshot once targets have had a moment to appear
        self._take_phase_screenshots("combat", wait_s=1.0)

        self._clicked_targets.clear()
        # Poll interval based on reaction time — targets need to be noticed quickly
        poll_interval = max(0.05, (self.cfg.reaction_ms / 4) / 1000.0)

        # Track mid-combat screenshot (taken after first few clicks)
        _mid_shot_taken = False
        _clicks_for_mid_shot = 3

        while True:
            time.sleep(poll_interval)

            # Check scene via JS — reliable, no accessibility tree limitations
            scene = self.pw.eval(JS_GET_SCENE).strip()
            if scene == "recap":
                return
            if scene != "combat":
                # Still in countdown or transitioning; keep waiting
                continue

            # Get current target IDs via JS — SVG elements are invisible to snapshot
            raw = self.pw.eval(JS_GET_TARGETS).strip()
            try:
                target_ids: list[int] = json.loads(raw) if raw and raw != "null" else []
            except json.JSONDecodeError:
                continue

            for tid in target_ids:
                if tid in self._clicked_targets:
                    continue

                # CPS limit
                now = time.time()
                min_gap = 1.0 / self.cfg.max_cps
                if now - self._last_click_time < min_gap:
                    continue

                # Miss rate
                if random.random() < self.cfg.miss_rate:
                    emit({"event": "miss", "ts": int(time.time()), "target_id": tid, "reason": "miss_rate"})
                    self._clicked_targets.add(tid)
                    continue

                # Reaction delay with jitter
                base = self.cfg.reaction_ms / 1000.0
                delay = base + random.uniform(-base * self.cfg.jitter, base * self.cfg.jitter)
                delay = max(0.01, delay)
                time.sleep(delay)

                # Choose target ring based on accuracy
                points = self._choose_points()

                # Click via JS eval — the only reliable way to interact with SVG elements
                result = self.pw.eval(js_click_target(tid, points))
                result = result.strip().strip("'\"")

                self._clicked_targets.add(tid)
                self._last_click_time = time.time()

                if result.startswith("clicked:"):
                    actual_points = int(result.split(":")[1]) if ":" in result else points
                    self.score += actual_points
                    emit({
                        "event": "click",
                        "ts": int(time.time()),
                        "target_id": tid,
                        "points": actual_points,
                        "reaction_ms": int(delay * 1000),
                        "cumulative_score": self.score,
                    })

                    # Mid-combat screenshot: capture live game state with score on board
                    if not _mid_shot_taken and len(self._clicked_targets) >= _clicks_for_mid_shot:
                        _mid_shot_taken = True
                        self._take_phase_screenshots("combat_active", wait_s=0.2)
                else:
                    # Target disappeared before we could click (already claimed or expired)
                    emit({"event": "miss", "ts": int(time.time()), "target_id": tid, "reason": result})

    def _choose_points(self) -> int:
        acc = self.cfg.accuracy
        r = random.random()
        if r < acc:
            return 4
        elif r < acc + (1 - acc) * 0.5:
            return 3
        elif r < acc + (1 - acc) * 0.8:
            return 2
        else:
            return 1

    # ------------------------------------------------------------------
    # Phase: recap
    # ------------------------------------------------------------------

    def read_recap(self) -> list[dict]:
        # Screenshot the recap screen
        self._take_phase_screenshots("recap", wait_s=0.5)

        raw = self.pw.eval(JS_GET_RECAP).strip()
        try:
            result = json.loads(raw) if raw and raw not in ("null", "") else []
            # eval() already stripped outer quotes and unescaped internals
            if isinstance(result, list):
                return result
            # If we somehow got a string (double-decoded), parse again
            if isinstance(result, str):
                return json.loads(result)
        except (json.JSONDecodeError, TypeError):
            pass
        return []

    def play_again(self):
        refs = self.pw.snapshot_refs()
        again_ref = find_ref(refs, "Play Again") or find_ref(refs, "play", "again")
        if again_ref:
            self.pw.click(again_ref)
            time.sleep(1.5)
        else:
            raise RuntimeError("Cannot find 'Play Again' button on recap screen")

    def cleanup(self):
        try:
            self.pw.close()
        except Exception:
            pass


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def parse_args() -> BotConfig:
    parser = argparse.ArgumentParser(description="Click Trainer Bot")
    parser.add_argument("--server", default="http://localhost:8080")
    parser.add_argument("--room-code", default=None)
    parser.add_argument("--name", default=None)
    parser.add_argument("--skill", default="intermediate",
                        choices=list(SKILL_PRESETS.keys()))
    parser.add_argument("--reaction-ms", type=float, default=None)
    parser.add_argument("--miss-rate", type=float, default=None)
    parser.add_argument("--accuracy", type=float, default=None)
    parser.add_argument("--max-cps", type=float, default=None)
    parser.add_argument("--rounds", type=int, default=1)
    parser.add_argument("--verbose", action="store_true")
    parser.add_argument("--screenshots", action="store_true",
                        help="Capture desktop + portrait screenshots at key phases")
    parser.add_argument("--screenshot-prefix", default=None,
                        help="Filename prefix for screenshots (default: botname_timestamp)")
    args = parser.parse_args()

    preset = SKILL_PRESETS[args.skill]
    cfg = BotConfig(
        server=args.server,
        room_code=args.room_code.upper() if args.room_code else None,
        skill=args.skill,
        reaction_ms=args.reaction_ms if args.reaction_ms is not None else preset["reaction_ms"],
        jitter=preset["jitter"],
        accuracy=args.accuracy if args.accuracy is not None else preset["accuracy"],
        miss_rate=args.miss_rate if args.miss_rate is not None else preset["miss_rate"],
        max_cps=args.max_cps if args.max_cps is not None else preset["max_cps"],
        rounds=args.rounds,
        verbose=args.verbose,
        screenshots=args.screenshots,
    )

    suffix = "".join(random.choices(string.ascii_uppercase + string.digits, k=4))
    cfg.name = args.name or f"Bot-{suffix}"
    cfg.session = f"bot-{cfg.name.lower().replace(' ', '-')}-{suffix}"

    if args.screenshot_prefix:
        cfg.screenshot_prefix = args.screenshot_prefix
    else:
        ts = int(time.time())
        safe_name = re.sub(r'[^a-z0-9]+', '_', cfg.name.lower())
        cfg.screenshot_prefix = f"{safe_name}_{ts}"

    return cfg


def main():
    cfg = parse_args()
    bot = BotPlayer(cfg)

    if cfg.screenshots:
        os.makedirs("screenshots", exist_ok=True)
        print(f"[screenshots] enabled — prefix: {cfg.screenshot_prefix}", file=sys.stderr)

    try:
        bot.navigate()

        room_code = bot.create_or_join()
        bot.room_code = room_code

        bot.register()

        emit({
            "event": "registered",
            "player_id": bot.player_id,
            "name": cfg.name,
            "room_code": room_code,
        })

        for round_num in range(cfg.rounds):
            bot.lobby_ready()
            bot.wait_for_combat()

            # Re-attempt player_id extraction: scoreboard is now in the DOM
            if not bot.player_id:
                bot._extract_player_id()

            bot.score = 0
            bot.combat()

            players = bot.read_recap()
            emit({
                "event": "recap",
                "ts": int(time.time()),
                "round": round_num + 1,
                "final_score": bot.score,
                "players": players,
            })

            if round_num < cfg.rounds - 1:
                bot.play_again()

    except KeyboardInterrupt:
        emit({"event": "interrupted", "ts": int(time.time())})
    except Exception as e:
        emit({"event": "error", "ts": int(time.time()), "message": str(e)})
        if cfg.verbose:
            import traceback
            traceback.print_exc(file=sys.stderr)
        sys.exit(1)
    finally:
        bot.cleanup()
        bot.print_screenshot_summary()


if __name__ == "__main__":
    main()
