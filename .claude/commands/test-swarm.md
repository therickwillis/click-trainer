# /test-swarm — Multi-Bot Swarm

Launch multiple bots into the same room to test multiplayer functionality.

## Setup
```bash
curl -sf http://localhost:8080/health || (ROUND_DURATION=15 ./tmp/main.exe &> /tmp/server.log & sleep 2)
```

## Common Invocations

**2-player game (default stagger):**
```bash
python3 scripts/swarm.py --count 2 --skill elite --rounds 1 \
  --stagger-ms 1000
```
PASS: 2 `registered` events, 2 `recap` events with different scores.

**2-player with screenshots:**
```bash
python3 scripts/swarm.py --count 2 --skill elite --rounds 1 \
  --screenshots --screenshot-prefix swarm2 --stagger-ms 1000
```
PASS: screenshot events from bot 0 at lobby/combat/combat_active/recap phases.

**Mixed skills:**
```bash
python3 scripts/swarm.py --count 3 --skills elite,beginner,intermediate \
  --rounds 1 --stagger-ms 800
```
PASS: Recap scores show elite > intermediate > beginner ordering.

**Stress test (5 bots):**
```bash
python3 scripts/swarm.py --count 5 --skill intermediate --rounds 2 \
  --stagger-ms 500
```
PASS: All 5 bots complete both rounds, SSE broadcasts reach all players.

## Output
Swarm prints per-bot colored lines. Capture stdout for event parsing:
```bash
python3 scripts/swarm.py --count 2 --skill elite --rounds 1 2>/dev/null
```
Lines prefixed `[Bot0]`, `[Bot1]`, etc. Final output is score ranking table.

## Failure Summary
- [ ] S1: All bots emit `registered` events (all joined the room)
- [ ] S2: All bots emit `combat_start` (SSE scene transition reached everyone)
- [ ] S3: All bots emit at least 1 `click` (targets spawned for all)
- [ ] S4: All bots emit `recap` with final_score > 0
- [ ] S5: Score ranking in final output lists all bots in descending score order
- [ ] S6 (if --screenshots): screenshot events from bot 0
- [ ] S7: No `error` events from any bot
