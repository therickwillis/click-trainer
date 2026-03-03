# /test-ui — Visual UI Responsive Design Test

Run a bot through a complete game capturing all viewport presets, then visually
analyze each screenshot for layout integrity and responsive design quality.

## Setup
```bash
mkdir -p screenshots
curl -sf http://localhost:8080/health || (ROUND_DURATION=15 ./tmp/main.exe &> /tmp/server.log & sleep 2)
```

## Phase U1 — Capture screenshots across all viewports

```bash
python3 scripts/bot.py \
  --name UITest \
  --skill elite \
  --screenshots \
  --screenshot-prefix ui_test \
  --viewports desktop,tablet-landscape,tablet-portrait,mobile-portrait,mobile-landscape \
  --rounds 1
```

- **U1.1**: `registered` event with non-empty room_code — PASS/FAIL
- **U1.2**: `screenshot` events emitted (up to 20: 5 viewports × 4 phases) — PASS/FAIL
- **U1.3**: No `error` event — PASS/FAIL

## Phase U2 — Inventory screenshots

```bash
ls screenshots/ui_test_*.png
```

Expected files (20 total):
- Phases: `lobby`, `combat`, `combat_active`, `recap`
- Viewports: `desktop`, `tablet-landscape`, `tablet-portrait`, `mobile-portrait`, `mobile-landscape`

Note: `combat_active` requires ≥3 successful clicks before round ends — may be missing.

## Phase U3 — Visual analysis

Read each screenshot with the Read tool. Analyze in order: desktop (baseline), then
tablet-landscape, tablet-portrait, mobile-portrait, mobile-landscape.

For **each screenshot** check:

**Layout integrity**
- Game area visible and not clipped by frame?
- HUD (timer, scores, player chip) visible at top?
- Portrait viewports: game rotated -90°?
- Landscape non-desktop: game NOT rotated?
- Frame overlay present with correct color? (orange vw<768, blue vw<1280, none for desktop)
- Frame label shows correct WxH dimensions in corner?

**Content readability**
- Lobby: room code readable? Ready button visible?
- Combat: SVG targets visible? Timer legible?
- Recap: player names and scores readable? "Game Over!" heading visible?
- Any text clipped, overflowing, or unreadably small?

**HUD/scoreboard behavior**
- Desktop + tablet-landscape: player name chips visible?
- Portrait + mobile-landscape: player name chips hidden?
- Rank chip showing position and score?

**Cross-viewport degradation** (after all viewports for a phase):
- Does the UI degrade gracefully desktop → tablet → mobile?
- Any interactive elements obscured at smaller sizes?
- Scaling proportionate (not stretched/squashed)?

## Phase U4 — Summary report

### Screenshot Inventory
List each file: FOUND or MISSING.

### Per-Viewport Analysis
For each viewport, a table:

| Phase | Layout OK | Readable | HUD OK | Notes |
|-------|-----------|----------|--------|-------|
| lobby | | | | |
| combat | | | | |
| combat_active | | | | |
| recap | | | | |

### Cross-Viewport Degradation
- desktop → tablet-landscape: GRACEFUL / DEGRADED / BROKEN
- tablet-landscape → tablet-portrait: GRACEFUL / DEGRADED / BROKEN
- tablet-portrait → mobile-portrait: GRACEFUL / DEGRADED / BROKEN
- mobile-portrait → mobile-landscape: GRACEFUL / DEGRADED / BROKEN

### Issues Found
viewport | phase | description | severity (Low/Med/High) | file

### Overall Result
PASS / WARN (minor issues) / FAIL (critical issues)
