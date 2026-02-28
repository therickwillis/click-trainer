# /test-combat — Combat Overlay Test

**First, run the base skill in full:**

> Run the `/test-integrated` skill now. Execute every phase in full. Record all pass/fail results before proceeding to the combat phases below.

After the base skill completes, continue with the following combat-specific phases. Use separate sessions `c1` and `c2` so base-skill sessions remain isolated.

---

## Setup

```bash
mkdir -p screenshots
```

Create a fresh two-player combat room for all combat phases:

```bash
# c1 creates room
playwright-cli -s=c1 open http://localhost:8080
playwright-cli -s=c1 click <create-room-ref>
```
Record the 4-char room code (e.g. `CMBT`).
```bash
playwright-cli -s=c1 snapshot
playwright-cli -s=c1 fill <name-input-ref> "Fighter1"
playwright-cli -s=c1 click <register-submit-ref>

# c2 joins
playwright-cli -s=c2 open http://localhost:8080
playwright-cli -s=c2 snapshot
playwright-cli -s=c2 fill <code-input-ref> "<CMBT>"
playwright-cli -s=c2 click <join-submit-ref>
playwright-cli -s=c2 snapshot
playwright-cli -s=c2 fill <name-input-ref> "Fighter2"
playwright-cli -s=c2 click <register-submit-ref>

# Both ready up
playwright-cli -s=c1 snapshot
playwright-cli -s=c1 click <ready-button-ref>
playwright-cli -s=c2 snapshot
playwright-cli -s=c2 click <ready-button-ref>
sleep 5
```

**PASS (setup)**: Both sessions show `data-scene="combat"` on the body element and `id="game-area"` is present.

---

## Phase C1 — Target spawn verification

**C1.1** Take initial snapshot and verify target count:
```bash
playwright-cli -s=c1 screenshot --filename=screenshots/c1-01-initial-targets.png
playwright-cli -s=c1 snapshot
```
**PASS**: Exactly 3 target divs (`id="target_1"`, `id="target_2"`, `id="target_3"` or consecutive IDs) are present inside `#targets`. This matches `InitialTargets: 3` in `gameCfg`.

**C1.2** Verify all 4 point zones on a target SVG:
```bash
playwright-cli -s=c1 snapshot
```
**PASS**: Within any target div (`data-target-id` attribute present), the SVG contains exactly 4 `circle` elements with `data-points="1"`, `data-points="2"`, `data-points="3"`, and `data-points="4"`. All 4 are visible.

**C1.3** Click one target and verify respawn timing:
Record the current list of target IDs in `#targets`. Click any target circle.
```bash
playwright-cli -s=c1 click <any-target-circle-ref>
playwright-cli -s=c1 screenshot --filename=screenshots/c1-02-target-clicked.png
sleep 2
playwright-cli -s=c1 screenshot --filename=screenshots/c1-03-target-respawned.png
playwright-cli -s=c1 snapshot
```
**PASS**: The clicked target's div is removed from the DOM immediately after click. After ≤1 second, a new target div appears inside `#targets` via the SSE `newTarget` event. Total target count returns to 3. (Respawn uses `time.AfterFunc(500ms, ...)` server-side.)

---

## Phase C2 — Score update verification

**C2.1** Outer ring awards 1 point:
Before clicking, record the current value shown in `id="player_score_{Fighter1_ID}"` (should be ≥ 0 from any earlier clicks).
```bash
playwright-cli -s=c1 snapshot
```
Find a target and click the outermost circle (`data-points="1"`):
```bash
playwright-cli -s=c1 click <outer-ring-data-points-1-ref>
playwright-cli -s=c1 snapshot
```
**PASS**: The `id="player_score_{Fighter1_ID}"` value increases by exactly 1. The `id="my_rank_score_{Fighter1_ID}"` span also shows the updated score.

**C2.2** Bullseye awards 4 points:
```bash
playwright-cli -s=c1 snapshot
```
Click the innermost circle (`data-points="4"`) on any available target:
```bash
playwright-cli -s=c1 click <inner-bullseye-data-points-4-ref>
playwright-cli -s=c1 snapshot
```
**PASS**: Score increases by exactly 4. If the bullseye ring is very small and hard to click precisely, clicking `data-points="3"` (inner ring, 3 pts) is also a valid proxy — verify the delta matches the `data-points` value of the ring clicked.

**C2.3** Cross-player score visibility via SSE:
After c1 clicks a target:
```bash
playwright-cli -s=c2 screenshot --filename=screenshots/c2-01-cross-score.png
playwright-cli -s=c2 snapshot
```
**PASS**: c2's `#scoreboard` shows Fighter1's score updated (the `id="player_score_{Fighter1_ID}"` element reflects the score from c1's click). The SSE `swap` broadcast propagated `player_score_*` and `my_rank_score_*` OOB updates to all subscribers.

**C2.4** Rank position updates:
```bash
playwright-cli -s=c1 snapshot
```
**PASS**: `id="my_rank_pos_{Fighter1_ID}"` shows `#1` or `#2` based on current scores. After c2 scores more than c1, c1's rank chip updates to `#2`.

---

## Phase C3 — Timer countdown accuracy

**C3.1** Sample timer values:
```bash
playwright-cli -s=c1 snapshot
```
Note the current value of `id="timer"` (e.g. `12`).
```bash
sleep 3
playwright-cli -s=c1 snapshot
```
**PASS**: The new value of `id="timer"` is at least 2 less than the recorded value (i.e., the timer decremented ≥2 over a 3-second window). This confirms `startRoundTimer` is ticking correctly with 1-second sleeps.

**C3.2** Recap appears at timer = 0:
```bash
sleep 17
playwright-cli -s=c1 screenshot --filename=screenshots/c1-04-recap.png
playwright-cli -s=c1 snapshot
```
**PASS**: `id="recap"` is present. Text "Game Over!" is visible. The `id="timer"` element is no longer present or shows `0`. Both Fighter1 and Fighter2 appear on the recap screen sorted by score descending.

---

## Phase C4 — Duplicate click protection

Start a fresh round to test this. After Phase C3 recap:
```bash
playwright-cli -s=c1 click <play-again-ref>
playwright-cli -s=c1 snapshot
playwright-cli -s=c1 click <ready-button-ref>
playwright-cli -s=c2 snapshot
playwright-cli -s=c2 click <ready-button-ref>
sleep 5
```
**PASS**: Both sessions are back in combat.

**C4.1** Rapid double-click on the same target:
```bash
playwright-cli -s=c1 snapshot
```
Record a specific `target_N` ID. Record the current score. Click that target, then immediately click on where it was (or attempt to click the same ref again):
```bash
playwright-cli -s=c1 click <target-N-circle-ref>
playwright-cli -s=c1 click <target-N-circle-ref>
sleep 1
playwright-cli -s=c1 snapshot
playwright-cli -s=c1 screenshot --filename=screenshots/c1-05-dedup.png
```
**PASS**: The score incremented only once (by the points value of the ring clicked). The second click on the dead target is silently ignored by `Targets.Kill()` returning `false`. No error is shown. A new target spawned (replacing the dead one) within ~1 second.

---

## Phase C5 — Failure summary (combat + base)

Report a unified checklist merging base results and combat results:

```
=== BASE SKILL RESULTS (from /test-integrated) ===
[paste base checklist here with PASS/FAIL per step]

=== COMBAT PHASES ===
[ ] Setup: Two-player room enters combat (data-scene=combat on both sessions)

C1 — Target spawn:
[ ] C1.1 Exactly 3 initial targets in #targets
[ ] C1.2 All 4 data-points circles (1–4) present on each target SVG
[ ] C1.3 Clicked target removed; new target spawns within 1s; count returns to 3

C2 — Score updates:
[ ] C2.1 Outer ring (data-points=1) → score +1 exactly
[ ] C2.2 Bullseye or inner ring → score +N matching data-points value
[ ] C2.3 Cross-player score visible via SSE on c2's #scoreboard
[ ] C2.4 my_rank_pos_{ID} and my_rank_score_{ID} update after clicks

C3 — Timer:
[ ] C3.1 Timer decrements ≥2 over 3-second window
[ ] C3.2 Recap appears when timer reaches 0

C4 — Duplicate click protection:
[ ] C4.1 Second click on dead target is no-op; score increments only once
```

Combat screenshots collected:
- screenshots/c1-01-initial-targets.png
- screenshots/c1-02-target-clicked.png
- screenshots/c1-03-target-respawned.png
- screenshots/c2-01-cross-score.png
- screenshots/c1-04-recap.png
- screenshots/c1-05-dedup.png

For every FAIL, include: phase ID, expected criterion, actual observed value, relevant screenshot.
