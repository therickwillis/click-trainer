# /test-integrated — Full Integrated Test

Run a complete build-and-verify cycle for Click Trainer. **Never stop early.** Collect every failure and report all of them in Phase 6. Treat each numbered step as a checkpoint with an explicit pass criterion.

---

## Setup

Before any phase, ensure the `screenshots/` directory exists:
```bash
mkdir -p screenshots
```

---

## Phase 1 — Pre-flight checks

**1.1** Build the binary:
```bash
go build -o ./tmp/main.exe ./cmd/web
```
**PASS**: exit code 0, no error output.

**1.2** Run the test suite:
```bash
go test ./...
```
**PASS**: output contains no lines beginning with `FAIL`.

**1.3** Run the linter:
```bash
golangci-lint run ./...
```
**PASS**: no output (exit 0). Record any output as failures but continue.

---

## Phase 2 — Server setup

**2.1** Check if the server is already running:
```bash
curl -sf http://localhost:8080/health
```

If the above fails, start the server with a short round duration:
```bash
ROUND_DURATION=15 ./tmp/main.exe &
sleep 2
curl -sf http://localhost:8080/health
```
**PASS**: response body is exactly `{"status":"ok"}` (HTTP 200).

---

## Phase 3 — Single-player happy path (`scripts/bot.py`)

Run the bot for 2 rounds with `--screenshots` so it captures lobby, combat, and recap phases automatically. The bot uses JS eval to click SVG targets — the only reliable approach for these elements.

```bash
python3 scripts/bot.py \
  --name Solo \
  --skill elite \
  --screenshots \
  --screenshot-prefix sp1 \
  --rounds 2
```

Collect the JSON event stream from stdout. Each line is a JSON object.

**3.1** Home page — **PASS** if the bot successfully navigates and emits a `registered` event (the bot opens the home page, finds Create Room, and proceeds through the join form).

**3.2** Create room — **PASS** if the `registered` event contains a `room_code` matching `^[A-Z0-9]{4}$`.

**3.3** Register "Solo" — **PASS** if the `registered` event is emitted with `"name": "Solo"` and a non-empty `player_id`.

**3.4** Ready up / combat start — **PASS** if a `combat_start` event is emitted (bot polled `data-scene` until it equalled `"combat"`).

**3.5** Click target, score > 0 — **PASS** if at least one `click` event is emitted with `"points": N` where N > 0. The bot dispatches `mousedown` on `circle[data-target-id]` elements via JS eval — confirms target removal and scoring.

**3.6** Round ends → recap — **PASS** if a `recap` event is emitted for `"round": 1`.

**3.7** Play again → lobby reset — **PASS** if a second `combat_start` event is emitted for round 2 (bot clicked Play Again, saw lobby with score reset, clicked Ready, and entered combat again). A second `recap` event for `"round": 2` confirms full cycle.

Screenshots captured by the bot (desktop + portrait-sim pair per phase):
- `screenshots/sp1_lobby_desktop.png` — lobby before round 1 ready
- `screenshots/sp1_combat_desktop.png` — combat start, round 1
- `screenshots/sp1_combat_active_desktop.png` — mid-combat after first clicks
- `screenshots/sp1_recap_desktop.png` — recap, round 1
- `screenshots/sp1_lobby_desktop.png` — lobby after play again (round 2)
- `screenshots/sp1_recap_desktop.png` — recap, round 2

---

## Phase 4 — Two-player happy path (`scripts/swarm.py`)

The swarm script handles bot coordination: bot 0 creates the room, followers wait for the room code via an internal event, and all bots poll `.lobby-player` count before clicking ready (enforced by `--min-players`, which swarm passes automatically).

```bash
python3 scripts/swarm.py \
  --count 2 \
  --skill elite \
  --rounds 1 \
  --screenshots \
  --screenshot-prefix swarm2 \
  --stagger-ms 1000
```

Read the swarm's coloured output. Each line is prefixed with the bot name.

**4.1** P1 creates room + registers — **PASS** if `[Bot1-*] joined room XXXX as Bot1-*` appears.

**4.2** P2 joins + registers — **PASS** if `[Bot2-*] joined room XXXX as Bot2-*` appears **with the same room code** as Bot1.

**4.3** P1 sees P2 via SSE — **PASS** implied: both bots poll `document.querySelectorAll('.lobby-player').length >= 2` before clicking ready. If this condition is never met, the bots time out and fail — so a successful combat start proves the SSE `newPlayer` broadcast delivered P2's card to Bot1's DOM.

**4.4** Both ready, combat starts — **PASS** if `[Bot1-*] combat started` and `[Bot2-*] combat started` both appear in the output.

**4.5** Cross-player scores via SSE — **PASS** if both bots emit `click` events with different `target_id` values. Targets are server-assigned per-room, so both bots operating on distinct IDs in the same room confirms the SSE score-broadcast pipeline is live.

**4.6** Recap, sorted by score — **PASS** if the `--- Final Scores ---` block at swarm exit shows Bot scores in descending order (highest first).

Screenshots captured by swarm (bot 0 only, desktop + portrait-sim):
- `screenshots/swarm2_lobby_desktop.png`
- `screenshots/swarm2_combat_desktop.png`
- `screenshots/swarm2_recap_desktop.png`

---

## Phase 5 — Edge cases (`scripts/test_edge.py`)

Run the automated edge-case script. It manages its own browser sessions and screenshots internally:

```bash
python3 scripts/test_edge.py
```

The script tests three scenarios and exits 0 on full pass, 1 on any failure.

**PASS**: Output contains `9 passed, 0 failed`. Each check line starts with `PASS:`:

```
--- E1: Invalid room code (ZZZZ) ---
  PASS: E1.1: URL does not contain /room/ after bad join
  PASS: E1.2: .error-msg element exists on page
  PASS: E1.3: .error-msg text is 'Room not found'

--- E2: Last player leaves → room deleted ---
  PASS: E2.1: Leaving redirects creator away from /room/
  PASS: E2.2: Accessing deleted room redirects visitor away from /room/

--- E3: Deep link without session cookie ---
  PASS: E3.1: Deep link keeps URL on /room/{code} (no redirect to /)
  PASS: E3.2: Page title indicates join/register template
  PASS: E3.3: Name input field is present (register form rendered)
  PASS: E3.4: #lobby element is NOT present (player not yet registered)

=== Results: 9 passed, 0 failed ===
```

Screenshots saved by the script:
- `screenshots/edge_e1_invalid_code.png`
- `screenshots/edge_e2_after_leave.png`
- `screenshots/edge_e2_deleted_room.png`
- `screenshots/edge_e3_deep_link.png`

Map the 9 sub-checks to the three Phase 5 steps for the failure summary:
- **5.1** = E1.1 + E1.2 + E1.3
- **5.2** = E2.1 + E2.2
- **5.3** = E3.1 + E3.2 + E3.3 + E3.4

---

## Phase 6 — Failure summary

Compile a final checklist of all 20 steps with PASS/FAIL status:

```
Pre-flight:
[ ] 1.1 Build (go build exit 0)
[ ] 1.2 Tests (go test — no FAIL lines)
[ ] 1.3 Lint (golangci-lint — no output)

Server:
[ ] 2.1 Health endpoint returns {"status":"ok"}

Single-player:
[ ] 3.1 Home page — CLICK TRAINER heading, create + join forms
[ ] 3.2 Create room — 4-char code in URL
[ ] 3.3 Register Solo — lobby div, player card visible
[ ] 3.4 Ready up — countdown overlay then data-scene=combat
[ ] 3.5 Click target — target removed, score > 0, new target spawns
[ ] 3.6 Round ends — #recap with "Game Over!" and Solo on podium
[ ] 3.7 Play again — lobby resets, score = 0

Two-player:
[ ] 4.1 P1 creates room + registers as PlayerOne
[ ] 4.2 P2 joins + registers as PlayerTwo
[ ] 4.3 P1 sees PlayerTwo via SSE newPlayer broadcast
[ ] 4.4 Both ready — countdown triggers, combat starts
[ ] 4.5 Cross-player scores visible via SSE
[ ] 4.6 Recap shows both players sorted by score desc

Edge cases:
[ ] 5.1 Invalid code ZZZZ → class="error-msg" with "Room not found"
[ ] 5.2 Last player leaves → room deleted → redirect to /
[ ] 5.3 Deep link without session → join form renders
```

List all screenshots collected:

Single-player (bot.py, desktop + portrait-sim pairs):
- screenshots/sp1_lobby_desktop.png + sp1_lobby_portrait_sim.png (×2 — pre-round-1 and post-play-again)
- screenshots/sp1_combat_desktop.png + sp1_combat_portrait_sim.png
- screenshots/sp1_combat_active_desktop.png + sp1_combat_active_portrait_sim.png
- screenshots/sp1_recap_desktop.png + sp1_recap_portrait_sim.png (×2 — round 1 and round 2)

Two-player (swarm.py, bot 0 only, desktop + portrait-sim pairs):
- screenshots/swarm2_lobby_desktop.png + swarm2_lobby_portrait_sim.png
- screenshots/swarm2_combat_desktop.png + swarm2_combat_portrait_sim.png
- screenshots/swarm2_recap_desktop.png + swarm2_recap_portrait_sim.png

Edge cases (test_edge.py):
- screenshots/edge_e1_invalid_code.png
- screenshots/edge_e2_after_leave.png
- screenshots/edge_e2_deleted_room.png
- screenshots/edge_e3_deep_link.png

For every FAIL, include: step number, expected criterion, actual observed value, relevant screenshot filename.
