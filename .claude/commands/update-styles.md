# /update-styles — CSS Styling Change Guide

Use this skill whenever modifying `static/styles.css`, responsive layout in `templates/game.html`,
or viewport simulation in `scripts/bot.py`. It documents the architecture, gotchas, and a
post-change verification checklist specific to this project.

---

## Architecture Overview

### CSS Layer Order
```
@layer reset, tokens, base, background, layout, components, utilities;
```
Put new rules in the correct layer. `@layer` order determines cascade priority — later layers win.

### Key Files
| File | Role |
|------|------|
| `static/styles.css` | All styling: layers, container queries, animations |
| `templates/game.html` | Inline `<script>` (pre-render scale), `updateGameScale()` JS, `#game-overlay` div |
| `scripts/bot.py` | `js_inject_viewport()` + `JS_RESTORE_DESKTOP` — must mirror CSS layout |

---

## Responsive Design: Container Query Architecture

The game uses a fixed 600×400px logical coordinate space. Visual scaling is CSS-transform-based.

### Containers
```css
.game-layout {
  container-type: size;      /* queries: @container game-viewport (...) */
  container-name: game-viewport;
}
.game-hud {
  container-type: inline-size;   /* queries: @container game-hud (...) */
  container-name: game-hud;
}
```

### Active Breakpoints
| Query | What changes |
|-------|-------------|
| `@container game-viewport (width <= 767px)` | Mobile layout: HUD fixed top, game-area rotated/scaled |
| `@container game-viewport (width >= 1400px)` | Large screen: wider HUD gap |
| `@container game-viewport (height <= 480px)` | Short viewport: compact HUD/timer |
| `@container game-hud (width < 500px)` | Hide player name chips |
| `@container game-hud (width < 320px)` | Hide player name + score chips, shrink timer |

### CRITICAL: `container-type: size` implies `contain: layout`
`contain: layout` makes `.game-layout` a **containing block for all positioned descendants**
(both `position: absolute` and `position: fixed`). This is safe here because `.game-layout`
is `position: fixed; inset: 0` (= full viewport). Any new `position: fixed` element that
should be relative to the **viewport** (not `.game-layout`) must be placed **outside**
`.game-layout` in the DOM — as a sibling under `<body>`.

Examples already outside `.game-layout`: `#game-overlay`, `.mute-btn`, copy-toast.

---

## Game-Area Centering (Mobile)

Inside `@container game-viewport (width <= 767px)`, the game-area is centered like this:

```css
#game-area {
  position: fixed;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%) rotate(calc(var(--game-rotate, 0) * -90deg)) scale(var(--game-scale));
}
```

**Do NOT use `inset: 0; margin: auto`** — auto margins fail (or go negative/zero) when the
element's logical width (600px) exceeds the viewport width, shifting the game off to the right.
`top: 50%; left: 50%; translate(-50%, -50%)` centers correctly at any viewport size.

---

## Shake Animation: `translate` vs `transform`

`@keyframes shake` uses the **individual `translate` CSS property**, not the `transform` shorthand:

```css
@keyframes shake {
  0%, 100% { translate: 0 0; }
  25%       { translate: -3px 2px; }
  50%       { translate: 3px -2px; }
  75%       { translate: -2px -1px; }
}
```

Individual transform properties (`translate`, `rotate`, `scale`) **compose with** the `transform`
shorthand rather than overriding it. The `transform` shorthand overrides itself — so if the shake
target ever gains a `transform` for positioning, the shake would erase it.

The shake is applied to `#targets` (not `#game-area`) so it never conflicts with the
game-area's positioning transform.

---

## Hit Effects: `#game-overlay`

Point popups (`+N`) and burst particles are appended to `#game-overlay`, **not** `#game-area`.

```html
<!-- in game.html body — sibling of .game-layout, NOT inside it -->
<div id="game-overlay"></div>
```
```css
#game-overlay { position: fixed; inset: 0; pointer-events: none; z-index: 50; }
```

**Why**: `#game-area` on mobile has `transform: rotate(-90deg)`, which rotates its children.
Popups inside it would float sideways instead of upward. Using viewport-space `clientX/Y`
coordinates on the non-rotated overlay keeps popups always upright.

JS event handlers use:
```js
var overlay = document.getElementById('game-overlay') || gameArea;
spawnPointsPopup(overlay, e.clientX, e.clientY, points);
spawnBurstParticles(overlay, e.clientX, e.clientY);
```

---

## Background Layer Z-Index

Always maintain explicit z-index order on `.scene-bg` children to prevent GPU compositor
layer reordering (which can make hills render behind sky):

```css
.bg-sky         { z-index: 1; }
.bg-stars       { z-index: 2; }
.bg-clouds-far  { z-index: 3; }
.bg-clouds-near { z-index: 4; }
.bg-hills       { z-index: 5; }
```

---

## Scale Computation (game.html)

Two places compute scale — both must stay in sync:

**Head inline script** (pre-render, no DOM access):
```js
var vh = window.visualViewport ? window.visualViewport.height : window.innerHeight;
// Desktop: scale = Math.min(vw * 0.95 / 600, vh * 0.95 / 452)  [452 = 400 + 52px HUD]
// Mobile portrait: scale = Math.min(vw * 0.95 / 400, (vh - hudH) * 0.95 / 600)
scale = Math.max(0.1, Math.min(scale, 8));  // always clamp
```

**`updateGameScale()` function** (post-render, has DOM):
- Desktop: uses `hud.scrollHeight || 52` for accurate HUD height
- Mobile: uses `hud.getBoundingClientRect().height` (HUD not scaled, so BBox is accurate)
- Also listens on `window.visualViewport` resize (excludes mobile keyboard / URL bar)

---

## bot.py Sync Requirement

**Whenever CSS layout for `#game-area` or scale formula changes, update `bot.py` to match.**

`js_inject_viewport()` and `JS_RESTORE_DESKTOP` manually mirror the CSS for screenshot viewports.
Current values (must stay in sync):

```python
HUD_H = 52                       # matches game.html HUD estimate
# scale factor: 0.95             # matches updateGameScale
# #game-area: top/left 50%       # NOT inset/margin
# transform: translate(-50%,-50%) rotate(...) scale(...)
# JS_RESTORE_DESKTOP: clears top/left (not inset/margin)
```

---

## Post-Change Verification Checklist

### Quick smoke test
```bash
python3 scripts/bot.py --name StyleCheck --skill elite --rounds 1
```
PASS: `registered`, `combat_start`, ≥1 `click`, `recap` — no `error`.

### Visual test across all viewports
```bash
python3 scripts/bot.py --name StyleCheck --skill elite --screenshots \
  --screenshot-prefix style_check \
  --viewports desktop,tablet-landscape,tablet-portrait,mobile-portrait,mobile-landscape \
  --rounds 1
```
Then use the Read tool to inspect screenshots. Check each viewport × phase:

| Check | desktop | tablet-ls | tablet-pt | mobile-pt | mobile-ls |
|-------|---------|-----------|-----------|-----------|-----------|
| Game area centered, not clipped | | | | | |
| HUD visible at top | | | | | |
| Portrait viewports: game rotated -90° | N/A | N/A | ✓ | ✓ | N/A |
| Hit popup text is upright (not rotated) | | | | | |
| Player name chips: visible at wide, hidden at narrow | | | | | |
| Background: hills above sky/clouds | | | | | |
| No content overflow or clipping | | | | | |

### CSS-specific checks
- New `@container` rules: test at the breakpoint boundary (±1px each side)
- New `z-index` changes: verify against the stacking context created by `contain: layout` on `.game-layout`
- New `position: fixed` elements: confirm they belong **outside** `.game-layout` in the DOM if they need viewport-relative placement
- Animation changes: verify `translate`/`rotate`/`scale` individual properties don't conflict with `transform` shorthand on the same element

### Full visual test
Run `/test-ui` for a complete responsive design audit with structured pass/fail output.
