# Skyvisor web — restyle specification

Date: 2026-09-03 · Specification only, no code changes · Builds on the existing token system in `app/assets/styles.css`

## 0. What this is

The current design system is not broken. It has an OKLCH token set with real light and dark values, a purpose-built map palette, deliberate reduced-motion handling, and code comments that explain why things are the way they are. This spec does not restart it. It sharpens a system that is already coherent and closes the places where the product's three faces do not yet look like one product.

## 1. Design direction

Skyvisor watches the sky and keeps the receipt. It tracks flights, it maps what is going wrong in the world, and it records the decisions an operations team made about both. The interface should feel like an operations position rather than a marketing site: legible under pressure, quiet when nothing is happening, unambiguous when something is. Dark mode is the hero, because the globe and the situation monitor are where the product is most itself, and because dark is the ambient condition of every room where this kind of screen actually lives.

The vernacular to draw on is aviation operations, not generic analytics. Flight progress strips, data blocks, timestamps that always carry a zone, source attribution next to every claim, status expressed as a state rather than a mood. The product's own positioning is that an agent may propose but a human decides, and the record survives. That means provenance is a first-class visual element: where a number came from, how old it is, and who acted on it belong on the surface, not behind a tooltip. This is also the honest differentiator, so it should be the thing the interface is most confident about.

What to avoid: the generic SaaS dashboard kit, where every piece of content becomes an identically rounded card with the same soft shadow and the same border radius regardless of importance. Avoid tracked-out all-caps eyebrow labels above every heading, meta strings joined by middle dots, arrows appended to button text, and gradient washes used as decoration. Avoid accenting one word of a headline in a different colour. The existing map chip and globe fade show the right instinct: a specific solution to a specific problem, explained. Extend that instinct; do not decorate around it.

## 2. Token deltas

All values are OKLCH, matching the existing convention. Add to `:root` and `.dark` in `app/assets/styles.css`, and expose through the `@theme inline` block the same way existing colours are.

### New semantic status tokens

The system currently has `--destructive` and nothing else for state. Situation severity is already modelled on the map (`--map-situation-watch`, `-warning`, `-severe`), but that vocabulary stops at the map edge. Bring it into the UI so a delayed flight, a degraded feed and a severe hazard read as the same scale everywhere.

| Token | Light | Dark |
|---|---|---|
| `--success` | `oklch(0.58 0.14 155)` | `oklch(0.72 0.15 158)` |
| `--success-foreground` | `oklch(0.99 0.005 155)` | `oklch(0.16 0.03 158)` |
| `--warning` | `oklch(0.68 0.15 70)` | `oklch(0.80 0.15 72)` |
| `--warning-foreground` | `oklch(0.20 0.04 70)` | `oklch(0.18 0.04 72)` |
| `--info` | `oklch(0.60 0.12 235)` | `oklch(0.75 0.12 236)` |
| `--info-foreground` | `oklch(0.99 0.005 235)` | `oklch(0.16 0.03 236)` |

Chroma is held deliberately below the primary blue so status never outshouts the brand, except `--destructive`, which should stay the loudest thing on any screen.

### Surface elevation

Today `--card` and `--popover` are the same value in both themes, so a sheet over a card over the page background is flat. Introduce an explicit three-step ramp; keep `--card` as-is so nothing existing shifts.

| Token | Light | Dark | Use |
|---|---|---|---|
| `--surface-1` | `oklch(1 0 0)` | `oklch(0.205 0.022 255)` | cards, the default raised plane (alias of today's `--card`) |
| `--surface-2` | `oklch(0.985 0.004 247)` | `oklch(0.235 0.023 255)` | nested panels, table headers, inset regions |
| `--surface-3` | `oklch(1 0 0)` | `oklch(0.265 0.024 255)` | popovers, sheets, dialogs, map overlay chrome |

In light mode the ramp is carried by borders rather than fills, which is why two of the three values coincide; that is intentional and should be commented in the stylesheet.

### Provenance tokens

Freshness and source attribution appear on the operations dashboard, the situation layers and every share and trust report. They currently borrow `--muted-foreground`, which makes stale data look like a caption.

| Token | Light | Dark |
|---|---|---|
| `--provenance-fresh` | `var(--success)` | `var(--success)` |
| `--provenance-aging` | `var(--warning)` | `var(--warning)` |
| `--provenance-stale` | `oklch(0.55 0.03 260)` | `oklch(0.62 0.03 260)` |

### Map palette

Two adjustments only. `--map-situation-advisory` and `--map-situation-news` are currently the same value (`oklch(0.78 0.14 250)`), so an advisory and a news pin are indistinguishable; move advisory to `oklch(0.80 0.10 220)`. `--map-situation-info` at chroma 0.05 is close to invisible against the dark ocean; lift to `oklch(0.75 0.07 260)`.

## 3. Typography

Keep Geist Variable. One family carries the whole interface; the monospace stack is not a second personality but a data convention.

| Role | Size / line-height | Weight | Tracking |
|---|---|---|---|
| Display (marketing hero only) | 3.25rem / 1.05 | 600 | -0.02em |
| Page title | 1.75rem / 1.2 | 600 | -0.015em |
| Section heading | 1.125rem / 1.35 | 600 | -0.01em |
| Body | 0.9375rem / 1.6 | 400 | 0 |
| Secondary / caption | 0.8125rem / 1.5 | 400 | 0 |
| Data label | 0.75rem / 1.4 | 500 | 0.01em |
| Metric value | 1.5rem / 1.1 | 600, tabular figures | -0.01em |

Monospace is reserved for identifiers that a person may need to read character by character or type back: flight numbers, ICAO and IATA codes, share tokens, case IDs, timestamps. It is not for generic small labels. Every numeric column and every metric value uses `font-variant-numeric: tabular-nums` so figures do not jitter as they update on a live surface.

Body copy caps at 72 characters. Sentence case everywhere, including buttons and table headers. No all-caps labels.

## 4. Spacing, radius, shadow

Spacing stays on Tailwind's 4px scale. Standardise three container rhythms: page gutter `1.5rem` mobile / `2rem` desktop, section gap `2.5rem`, card padding `1.25rem`.

Radius should encode hierarchy instead of being uniform. Today `--radius` feeds every size. Define the intent explicitly: `0.25rem` for inline chips and badges, `0.5rem` for controls and inputs, `0.75rem` for cards and panels, `1rem` for sheets and dialogs, full round for pills and avatars only. A map overlay panel and a form input should not share a corner.

Shadows carry elevation only, never decoration, and are near-invisible in dark mode where the surface ramp does that work instead. Two levels: a hairline `0 1px 2px` at 6% for raised cards, and `0 8px 24px` at 12% for overlays. Nothing else gets a shadow.

## 5. Per-surface changes

Priority: **P0** ships first, **P1** next, **P2** when convenient.

**`/track` and flight detail** — Currently a full-bleed MapLibre map with a progress strip and status sheet. The strongest available upgrade is to treat the status sheet as a flight data block: origin and destination as monospace codes at equal weight, the delta from schedule as the one loud number, and gate, aircraft and altitude as a quiet grid beneath. Route the delay figure through `--warning` and `--destructive` rather than colouring the whole card. Add a visible last-updated stamp fed by the provenance tokens. **P0.**

**`/globe`** — The globe already fades in only once it has drawn a frame, which is the right kind of detail. What it lacks is a resting state that explains itself: with no flight selected the page should say what is being shown and how current it is, rather than presenting an unlabelled sphere. Move the layer controls into a single `--surface-3` panel with the new radius scale, and give every layer toggle its legend swatch inline so the map legend and the control are the same object. **P0.**

**`/situation`** — This is the best screenshot in the product and should be treated as the flagship. Apply the severity scale consistently between the map pins and the news rail so a severe item reads identically in both. Each item carries its source and age using the provenance tokens; unverified layers say so plainly rather than being hidden. The rail should be scannable at a glance: severity swatch, place, one line, age. **P0.**

**`/dashboard` (operations)** — This is the surface that carries the actual business, and it currently looks the most like a generic analytics page. Rebuild the KPI row as metric tiles with tabular figures, an explicit sample size, and a freshness stamp on each. The decision loop is the differentiator, so the audit trail should be visible on the dashboard rather than one click away: last decisions, who approved, what the outcome was. **P0.**

**`/watches`** — The largest template in the codebase at 342 lines. Split the row into a dedicated component, apply status tokens to the state chip, and add a genuine empty state that invites the first action instead of reporting emptiness. **P1.**

**`/pricing`** — Must stop implying Business is self-serve while the webhook grants Pro; that is a correctness issue the design should not paper over. Present Free and Pro as purchasable, Business and Enterprise as a conversation, without visual apology. **P1.**

**Onboarding welcome** — One screen, one action. Ask nothing that can be inferred. Its job is to get to a first tracked flight. **P1.**

**Navbar and sidebar** — Unify the active state on one treatment across both. Currently the two chrome components carry different active affordances. **P2.**

## 6. Components

Keep and restyle the existing templUI set: `button`, `card`, `badge`, `input`, `dialog`, `popover`, `sheet`, `skeleton`, `toast`, `tooltip`, `separator`, `icon`, `aspectratio`.

Add, in this order:

1. **MetricTile** — value, label, delta, sample size, freshness stamp. The workhorse of the dashboard.
2. **StatusChip** — one component consuming the success/warning/destructive scale, used for flight state, feed health and case state alike.
3. **ProvenanceNote** — source name, retrieval time, verified or unverified. Sits under any surfaced claim.
4. **RatioMeter** — a bounded proportion with its denominator visible.
5. **EmptyState** — illustration slot, one sentence, one action.
6. **SourceBadge** — compact attribution for map and rail items.

`ionicons` currently ships 50 KB of icon CSS globally. Inline the roughly two dozen glyphs actually used as SVG and drop the dependency.

## 7. Motion

Motion One is already in place and every helper checks `reducedMotion()`. Three rules to add:

Animation runs once per element. The counter helper already guards on a dataset flag; the enter, progress and pulse helpers must do the same, so an htmx swap never replays an animation on content that did not change. This is finding W4 in the web audit and is the only motion defect worth fixing.

Durations: 120ms for state changes on controls, 240ms for entering surfaces, 400ms for a page-level orchestrated reveal. Nothing runs longer than 400ms except the deliberate 16s route drift, which is ambient rather than an entrance.

Continuous animation is reserved for things that are genuinely live: the route dash drift and the progress strip. A pulsing metric on a static dashboard is decoration and should be removed.

## 8. Dark and light parity

Every token defined in `:root` must have a `.dark` counterpart; the three new status pairs and the surface ramp above are specified for both. Check in both themes: `--muted-foreground` on `--surface-2` must clear 4.5:1, status chips must clear 4.5:1 against their own backgrounds, focus rings must be visible on both `--surface-1` and map overlays, and the map chip halo must hold up over both ocean and land. The globe and situation surfaces should be reviewed in light mode specifically, since they are designed dark-first and are the most likely to have been left behind.

## 9. Responsive

Mobile first, since the mobile web view is the near-term mobile story while the native app has no shippable target yet. Breakpoints stay Tailwind's defaults. Rules: the map is full-bleed below `md` with controls collapsed into a single sheet; tables become stacked records below `md` rather than scrolling horizontally, except the analytics export view where a horizontal scroll container is correct; the navbar collapses to a sheet below `lg`; touch targets are never under 44px; no surface may cause horizontal page scroll, only its own overflow container.

## 10. Implementation order

Each phase is independently shippable.

| Phase | Content | Effort |
|---|---|---|
| 1 | Token additions: status, surface ramp, provenance, the two map fixes. Radius scale. No visual change beyond what phase 2 consumes. | S |
| 2 | StatusChip, MetricTile, ProvenanceNote, EmptyState. Motion ready-guards (audit W4). | M |
| 3 | `/situation` and `/globe` restyle. The flagship screenshots. | M |
| 4 | `/dashboard` operations rebuild including the visible audit trail. | L |
| 5 | `/track` flight data block; `/watches` row component and empty state. | M |
| 6 | Pricing honesty pass, onboarding, nav unification, icon subsetting. | M |

Phases 1 and 2 are prerequisites for everything after them. Phase 3 is the one to do first if only one phase gets done, because the situation monitor is the most demonstrable surface in the product.
