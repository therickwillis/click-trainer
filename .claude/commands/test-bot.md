# /test-bot — Single Bot Player

Run a bot through the game. Useful for quick smoke testing or screenshot capture.

## Setup
```bash
curl -sf http://localhost:8080/health || (ROUND_DURATION=15 ./tmp/main.exe &> /tmp/server.log & sleep 2)
```

## Common Invocations

**Quick smoke test (1 round, elite skill):**
```bash
python3 scripts/bot.py --name Smokey --skill elite --rounds 1
```
PASS: `registered` event with room_code, `recap` event with final_score > 0.

**With screenshots (desktop + mobile-portrait):**
```bash
python3 scripts/bot.py --name SnapBot --skill elite --screenshots \
  --screenshot-prefix snap1 --rounds 1
```
PASS: `screenshot` events emitted for lobby, combat, combat_active, recap phases.

**Custom viewports:**
```bash
python3 scripts/bot.py --name UIBot --skill elite --screenshots \
  --screenshot-prefix custom1 \
  --viewports desktop,tablet-portrait,mobile-portrait --rounds 1
```

**Multiple rounds:**
```bash
python3 scripts/bot.py --name Grinder --skill intermediate --rounds 3
```
PASS: 3 `recap` events emitted.

## Output Events
Parse stdout as newline-delimited JSON:
- `registered` — player joined, has room_code and player_id
- `combat_start` — round started
- `click` — target hit with points and reaction_ms
- `miss` — target missed (miss_rate or target disappeared)
- `screenshot` — file captured with phase, viewport, dimensions
- `recap` — round ended with final_score and players array
- `error` — fatal error; check message field

## Failure Summary
- [ ] B1: `registered` event emitted
- [ ] B2: `combat_start` event emitted
- [ ] B3: At least 1 `click` event (targets were found and clicked)
- [ ] B4: `recap` event emitted with final_score > 0
- [ ] B5 (if --screenshots): screenshot events for all phases × viewports
- [ ] B6: No `error` event
