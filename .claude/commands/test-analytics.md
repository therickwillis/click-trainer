# /test-analytics — Analytics Overlay Test

**First, run the base skill in full:**

> Run the `/test-integrated` skill now. Execute every phase in full. Record all pass/fail results before proceeding to the analytics phases below.

After the base skill completes, continue with the following analytics-specific phases. Use session `an1` for all analytics browser interactions.

---

## Setup

```bash
mkdir -p screenshots
```

---

## Phase A0 — DB prerequisite check

**A0.1** Probe the analytics endpoint:
```bash
STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/analytics)
echo "Analytics status: $STATUS"
```

**If STATUS = 503**:
- **PASS (INFO)**: Server is running without a `DATABASE_URL`. Analytics requires a database.
- Mark all remaining analytics steps (A1–A5) as `SKIP — no database`. Add a single INFO entry to the failure summary.
- Jump directly to Phase A6 (failure summary).

**If STATUS = 200**:
- **PASS**: Database is connected. Proceed with A1–A5.

---

## Phase A1 — Dashboard loads and leaderboard tabs work

**A1.1** Open analytics dashboard:
```bash
playwright-cli -s=an1 open http://localhost:8080/analytics
playwright-cli -s=an1 screenshot --filename=screenshots/an1-01-dashboard.png
playwright-cli -s=an1 snapshot
```
**PASS**: Page renders without error. Text "Analytics" (`<h1>`) is visible. `id="leaderboard-content"` is present in the DOM. A leaderboard table or entry list is displayed (may be empty if no games have been played yet).

**A1.2** Click all 4 leaderboard tab buttons and verify `#leaderboard-content` updates:

Score tab:
```bash
playwright-cli -s=an1 click <score-tab-ref>
playwright-cli -s=an1 screenshot --filename=screenshots/an1-02-lb-score.png
playwright-cli -s=an1 snapshot
```
**PASS**: `#leaderboard-content` innerHTML was replaced via HTMX (`hx-target="#leaderboard-content" hx-swap="innerHTML"`). No JS error visible.

Wins tab:
```bash
playwright-cli -s=an1 click <wins-tab-ref>
playwright-cli -s=an1 screenshot --filename=screenshots/an1-03-lb-wins.png
```
**PASS**: `#leaderboard-content` updates.

Reaction tab:
```bash
playwright-cli -s=an1 click <reaction-tab-ref>
playwright-cli -s=an1 screenshot --filename=screenshots/an1-04-lb-reaction.png
```
**PASS**: `#leaderboard-content` updates.

Bullseyes tab:
```bash
playwright-cli -s=an1 click <bullseyes-tab-ref>
playwright-cli -s=an1 screenshot --filename=screenshots/an1-05-lb-bullseyes.png
```
**PASS**: `#leaderboard-content` updates.

**A1.3** Score leaderboard order:
```bash
playwright-cli -s=an1 click <score-tab-ref>
playwright-cli -s=an1 snapshot
```
**PASS**: If the leaderboard has ≥2 entries, the first entry's score value is ≥ the second entry's score value (descending order — confirmed by `ORDER BY value DESC` in `GetLeaderboard("score", ...)`).

---

## Phase A2 — Player stats on dashboard

**A2.1** Dashboard shows player stats when logged in:

The base skill registered players with `player_id` cookies. Open the dashboard from a session that has a valid `player_id` cookie (reuse an active Playwright session if possible, or use `an1` which may have a cookie from the base skill's game flow):
```bash
playwright-cli -s=an1 open http://localhost:8080/analytics
playwright-cli -s=an1 snapshot
playwright-cli -s=an1 screenshot --filename=screenshots/an1-06-player-stats.png
```
**PASS (conditional)**: If the `player_id` cookie matches a player in the DB, a "Your Stats" card is visible with `GamesPlayed`, `TotalScore`, `WinCount`, `BestGame` values. If no matching cookie, the stats card is absent — mark as INFO (expected when testing without prior games in this session).

**A2.2** Badge display after a completed game:
```bash
playwright-cli -s=an1 snapshot
```
**PASS (conditional)**: If any badge was earned during the base skill's game runs, badge elements (`<span class="analytics-badge">`) appear in the "Badges" section. Expected badges with short ROUND_DURATION=15:
- **Trigger Happy** (`🖱️`) — 3+ CPS is achievable in 15s rounds.
- **Speed Demon** (`⚡`) — <300ms avg reaction is achievable.
- **Perfectionist** (`✨`) — 50%+ bullseye rate is achievable.
- **Centurion** (`💯`) — 100+ pts requires ~34 clicks at 3pts avg in 15s; mark as INFO if not earned.
- **Sharpshooter** (`🎯`) — 10+ bullseyes in one game; mark as INFO if not earned.
- **Veteran** (`🏅`) — requires 10+ games; mark as INFO if not earned.
- **Unstoppable** (`🔥`) — requires 3-game win streak; mark as INFO if not earned.

---

## Phase A3 — Player profile page

**A3.1** Navigate to a player profile:

Get the player ID from the `player_id` cookie in an active session, or extract it from the leaderboard link:
```bash
PLAYER_ID=$(playwright-cli -s=an1 snapshot | grep -oP '(?<=/analytics/player/)[a-f0-9-]+' | head -1)
echo "Player ID: $PLAYER_ID"
```
If found:
```bash
playwright-cli -s=an1 open http://localhost:8080/analytics/player/$PLAYER_ID
playwright-cli -s=an1 screenshot --filename=screenshots/an1-07-player-profile.png
playwright-cli -s=an1 snapshot
```
**PASS**: Page renders the `analytics-player` template. `GamesPlayed` value > 0. `BestGame` value > 0. Player name is visible.

**A3.2** Dashboard → player profile link navigation:
From the dashboard, if player links are present in the leaderboard:
```bash
playwright-cli -s=an1 open http://localhost:8080/analytics
playwright-cli -s=an1 snapshot
playwright-cli -s=an1 click <leaderboard-player-link-ref>
playwright-cli -s=an1 screenshot --filename=screenshots/an1-08-profile-via-link.png
```
**PASS**: Navigates to `/analytics/player/{id}` and the player profile template renders.

---

## Phase A4 — Game recap page

**A4.1** Navigate to a valid game recap:

Extract a game ID from a previously completed game. The server logs the game ID when `CreateGame` succeeds. Alternatively, check if the analytics dashboard leaderboard links include game IDs. If no game ID is readily available, complete one more round via Playwright and retrieve it from server logs or the analytics API:
```bash
GAME_ID=$(curl -s http://localhost:8080/analytics/leaderboard?cat=score | grep -oP '(?<=/analytics/game/)[a-f0-9-]+' | head -1)
echo "Game ID: $GAME_ID"
```
If found:
```bash
playwright-cli -s=an1 open http://localhost:8080/analytics/game/$GAME_ID
playwright-cli -s=an1 screenshot --filename=screenshots/an1-09-game-recap.png
playwright-cli -s=an1 snapshot
```
**PASS**: Page renders the `analytics-game` template. All participants are listed with their scores. CPS and bullseye rate columns are present. The game's room code and timestamps are shown.

**A4.2** Invalid game ID → 404:
```bash
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/analytics/game/00000000-0000-0000-0000-000000000000)
echo "Status: $HTTP_STATUS"
```
**PASS**: HTTP status is `404`. Response body contains "Game not found".

---

## Phase A5 — Leaderboard ordering and error handling

**A5.1** Score leaderboard is descending:
```bash
curl -s http://localhost:8080/analytics/leaderboard?cat=score
```
**PASS**: If ≥2 entries are returned, the first entry's numeric value ≥ second entry's value (descending — `ORDER BY value DESC`).

**A5.2** Reaction leaderboard is ascending (best/lowest reaction time first):
```bash
curl -s http://localhost:8080/analytics/leaderboard?cat=reaction
```
**PASS**: If ≥2 entries are returned, the first entry's reaction_ms value ≤ second entry's value (ascending — `ORDER BY value ASC`).

**A5.3** Unknown leaderboard category returns 500:
```bash
HTTP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/analytics/leaderboard?cat=invalid_category")
echo "Status: $HTTP_STATUS"
```
**PASS**: HTTP status is `500`. The `GetLeaderboard` function returns `fmt.Errorf("unknown leaderboard category: %s", category)` for unrecognized categories, which the handler converts to a 500 response.

**A5.4** Wins and bullseyes leaderboards:
```bash
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/analytics/leaderboard?cat=wins"
curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/analytics/leaderboard?cat=bullseyes"
```
**PASS**: Both return HTTP 200.

---

## Phase A6 — Failure summary (analytics + base)

Report a unified checklist merging base results and analytics results:

```
=== BASE SKILL RESULTS (from /test-integrated) ===
[paste base checklist here with PASS/FAIL per step]

=== ANALYTICS PHASES ===

A0 — DB prerequisite:
[ ] A0.1 GET /analytics → 200 (DB connected) or 503 (no DB — remaining steps SKIP)

A1 — Dashboard and leaderboard tabs (skip if A0.1 = 503):
[ ] A1.1 Dashboard loads — "Analytics" heading, #leaderboard-content present
[ ] A1.2a Score tab → #leaderboard-content updates
[ ] A1.2b Wins tab → #leaderboard-content updates
[ ] A1.2c Reaction tab → #leaderboard-content updates
[ ] A1.2d Bullseyes tab → #leaderboard-content updates
[ ] A1.3 Score leaderboard entries in descending order

A2 — Player stats and badges (skip if A0.1 = 503):
[ ] A2.1 Dashboard shows "Your Stats" card when player_id cookie valid (or INFO if no session)
[ ] A2.2 Badge elements visible after completed game (or INFO if no badges earned)
     INFO: Centurion (100+ pts in 15s) — INFO if not earned
     INFO: Sharpshooter (10+ bullseyes) — INFO if not earned
     INFO: Veteran (10+ games) — INFO if not earned
     INFO: Unstoppable (3-game win streak) — INFO if not earned

A3 — Player profile (skip if A0.1 = 503):
[ ] A3.1 /analytics/player/{id} renders with GamesPlayed > 0, BestGame > 0
[ ] A3.2 Dashboard leaderboard link navigates to player profile

A4 — Game recap (skip if A0.1 = 503):
[ ] A4.1 /analytics/game/{id} renders with participants, scores, CPS, bullseye rate
[ ] A4.2 Invalid game ID → HTTP 404 "Game not found"

A5 — Leaderboard ordering and error handling (skip if A0.1 = 503):
[ ] A5.1 Score leaderboard descending order
[ ] A5.2 Reaction leaderboard ascending order (best reaction first)
[ ] A5.3 Unknown category → HTTP 500
[ ] A5.4 Wins and bullseyes categories → HTTP 200
```

Analytics screenshots collected:
- screenshots/an1-01-dashboard.png
- screenshots/an1-02-lb-score.png
- screenshots/an1-03-lb-wins.png
- screenshots/an1-04-lb-reaction.png
- screenshots/an1-05-lb-bullseyes.png
- screenshots/an1-06-player-stats.png
- screenshots/an1-07-player-profile.png
- screenshots/an1-08-profile-via-link.png
- screenshots/an1-09-game-recap.png

For every FAIL, include: phase ID, expected criterion, actual observed value, relevant screenshot or curl output. For every SKIP, note the reason (no DATABASE_URL). For every INFO, note that the condition was not triggered but is not a failure.
