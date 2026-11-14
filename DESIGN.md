# checkship design

This is the owner's design direction for the checkship dashboard (`cmd/serve/front`). It covers three things:

1. **The design system.** Palette, type, shape, layout and patterns.
2. **The app as it is today.** Every page, every component and how the code is written.
3. **The UI migration.** The UI library chosen (shadcn-svelte on Bits UI), the dependency versions and the order of the steps.

The palette below is fixed and carries over unchanged. When this file and the CSS disagree, this file states the intent and the CSS is the thing to fix.

Versions were checked against the npm registry on 2026-11-14. Re-run `npm view <package> version` before each migration step, because several of these packages ship weekly.

---

## 1. Principles

- **Flat and solid.** Solid colours only: no gradients, glows, glass or drop-shadow stacks. _Why: a deployment dashboard is read for its state (ready, failed, running), and flat colour keeps attention on that state rather than on chrome._
- **Content first, little chrome.** Grouping comes from headings, spacing and tonal fills (canvas, surface, sunken).
  - A card is used only where an entity is a unit you act on: an application, target, registry or environment.
  - That card has a `line` border and no shadow.
  - Forms, lists and details sit directly on the surface.
- **Full height.** The shell fills the window. No floating centred panels with empty canvas around them. The sign-in screen is two full-height columns too (§5, Auth layout).
- **State is colour, colour is state.** `success`, `warning`, `danger` and `record` only ever mean a state, never decoration. The accent marks the one primary action, the selection and focus.
- **No dividers.** No separator lines, hairlines, rules or borders between groups, rows, panes or sections. Grouping comes from spacing, the size and weight of headings, and tonal fills (`canvas`, `surface`, `sunken`, `raised`).
  - shadcn's `Separator` is never used, and `--border` and `--sidebar-border` are transparent.
  - The only outlines left are input fields, selects, unselected radios and switches (the `faint` border), and the keyboard focus ring, because those are controls, not dividers.
  - Floating panels (menus, selects, dialogs, the mobile sheet) lift off the page with their own tone and a soft shadow, never a line.
  - _Why: the owner asked for no divider lines anywhere._
- **Words over numbers that mean nothing.** If a value has nothing to compare against, say so ("Not configured", "No deployment yet"), never `0` or `-`.

## 2. Dials

| Dial   | Value | What it means here                                                                           |
| ------ | ----- | -------------------------------------------------------------------------------------------- |
| ENERGY | 1     | Neutral surfaces, one accent, colour reserved for state and the primary action.              |
| RHYTHM | 2     | A steady 6px spacing scale with predictable page structure (title, actions, sections).       |
| MOTION | 1     | State transitions, the navigation progress bar and the running spinner. Nothing else moves. |

---

## 3. Palette

These are the owner's exact values and they do not change. Every colour is a CSS custom property on `:root`. In the migrated app they are mapped into Tailwind through `@theme inline` (section 12). Components use tokens, never raw hex.

```css
:root {
  --canvas: #eef0f3;
  --surface: #ffffff;
  --sunken: #f5f6f8;
  --line: #dde1e7;
  --line-strong: #c6ccd5;
  --ink: #14161a;
  --muted: #5b6270;
  --faint: #8a919c;
  --accent: #2f5bea;
  --accent-hover: #2549c4;
  --accent-soft: #e6ecfd;
  --on-accent: #ffffff;
  --record: #d93a2b;
  --record-soft: #fbe7e5;
  --success: #1f8a4c;
  --success-soft: #e3f3ea;
  --warning: #a86a12;
  --warning-soft: #fbf0dc;
  --danger: #c4322b;
  --danger-soft: #fbe6e5;
}

@media (prefers-color-scheme: dark) {
  :root {
    --canvas: #0b0b0d;
    --surface: #131316;
    --sunken: #0f1012;
    --line: #222327;
    --line-strong: #32343a;
    --ink: #f1f1f3;
    --muted: #a8adb7;
    --faint: #7e838f;
    --accent: #7f95ff;
    --accent-hover: #98a9ff;
    --accent-soft: #1f2540;
    --on-accent: #0d1020;
    --record: #ff6f64;
    --record-soft: #3a1d1b;
    --success: #4fce87;
    --success-soft: #143022;
    --warning: #e7b45a;
    --warning-soft: #32260f;
    --danger: #ff6f64;
    --danger-soft: #3a1d1b;
  }
}
```

In the migrated app, the dark block is applied by the `.dark` class that `mode-watcher` sets on `<html>`, so a person can pick Light, Dark or System. System follows `prefers-color-scheme` exactly as above. Only the selector changes; the values stay the same.

### Brand colour

`--brand` is `#2f5bea` in light and `#7f95ff` in dark. It is used by the logo only and never follows the accent tone. Today it lives in `src/assets/styles.css` as `--co-brand`, and it equals the default blue accent by design.

### Accent tones

Blue is the default and the only tone checkship ships today. The other tones are kept for a future Settings › Interface › Accent option. Each overrides four tokens through `:root[data-accent-tone='...']`.

| Tone   | Mode  | `--accent` | `--accent-hover` | `--accent-soft` | `--on-accent` |
| ------ | ----- | ---------- | ---------------- | --------------- | ------------- |
| violet | light | #6148d8    | #5038b8          | #ece8ff         | #ffffff       |
| green  | light | #1f8a4c    | #267b47          | #e3f3ea         | #ffffff       |
| amber  | light | #ba7400    | #9f6600          | #f7edda         | #241603       |
| rose   | light | #c14574    | #a73b64          | #f9e7ee         | #ffffff       |
| violet | dark  | #a28fff    | #b7a8ff          | #30254b         | #0f0820       |
| green  | dark  | #4fce87    | #6fdaa0          | #153025         | #081d12       |
| amber  | dark  | #e7b45a    | #efc680          | #38280f         | #241603       |
| rose   | dark  | #ff83ad    | #ff9fc0          | #3c1d2b         | #260d17       |

Light amber is the only tone with dark text on the accent fill. _Why: white on #ba7400 is too weak, and #241603 reads clearly on it._

### Roles

- **Surfaces.** `canvas` is the app background, shared by the sidebar and the title bar so they read as one frame. `surface` is the workspace. `sunken` is for rows, wells, inputs and setting rows.
- **Raised, overlay and hover** are not new colours; they reuse palette values now that lines are gone (§1):
  - `raised` = the `line` value: secondary buttons, segmented-control tracks, count chips.
  - `overlay` = `surface` in light, `line` in dark: menus, selects and dialogs.
  - `hover` = `canvas` in light, `line-strong` in dark: the hover fill on rows, menu items and ghost buttons.
- **Text.** `ink` is primary text. `muted` is descriptions, metadata, timestamps and secondary labels.
- **`faint`** is for SVG line art, disabled glyphs, decorative strokes and **the border of form controls** (inputs, textareas, selects, checkboxes, switch tracks: the `input` token), where it gives the 3:1 a control needs (3.18:1 light, 4.88:1 dark). **It is never used for text.**
- **`accent`** marks the primary action, the selection and focus. There is one primary action per view: New application, Deploy, Save, Sign in.
- **Selected items.** A selected nav item or row is an `accent-soft` fill with `accent` text and icon, never a coloured strip on its edge.
- **State colours.** `success`, `warning`, `danger` and `record` are for state only. Section 11.4 maps every checkship state to one of them.

## 4. Contrast

Values are WCAG 2.x contrast ratios. AA is 4.5:1 for normal text, and 3:1 for large text (18px and up) and for non-text UI.

### Light

| Pairing                                   | Ratio                 | Result                  |
| ----------------------------------------- | --------------------- | ----------------------- |
| ink on surface / sunken / canvas          | 18.11 / 16.75 / 15.86 | AA                      |
| muted on surface / sunken / canvas        | 6.13 / 5.67 / 5.37    | AA                      |
| faint on surface / sunken / canvas        | 3.18 / 2.94 / 2.78    | Not text (see rules)    |
| accent on surface / canvas                | 5.52 / 4.83           | AA                      |
| on-accent on accent / accent-hover        | 5.52 / 7.42           | AA                      |
| accent on accent-soft                     | 4.67                  | AA                      |
| success on surface                        | 4.38                  | Accepted exception      |
| warning on surface                        | 4.43                  | Accepted exception      |
| success on success-soft                   | 3.81                  | Below AA                |
| warning on warning-soft                   | 3.93                  | Below AA                |
| danger on surface / danger-soft           | 5.47 / 4.57           | AA                      |
| record on surface                         | 4.57                  | AA                      |
| violet: white on #6148d8 / #5038b8        | 6.13 / 8.08           | AA                      |
| green: white on #1f8a4c / #267b47         | 4.38 / 5.24           | Accepted exception / AA |
| amber: #241603 on #ba7400 / #9f6600       | 4.69 / 3.67           | AA / Below AA           |
| amber #ba7400 as text on surface / canvas | 3.76 / 3.29           | Below AA                |
| rose: white on #c14574 / #a73b64          | 4.80 / 6.08           | AA                      |

### Dark

| Pairing                               | Ratio              | Result                       |
| ------------------------------------- | ------------------ | ---------------------------- |
| ink on surface / canvas               | 16.44 / 17.43      | AA                           |
| muted on surface / canvas             | 8.24 / 8.73        | AA                           |
| faint on surface / canvas             | 4.88 / 5.18        | AA (still not used for text) |
| accent on surface                     | 6.75               | AA                           |
| on-accent on accent / accent-hover    | 6.87 / 8.50        | AA                           |
| success / warning / danger on surface | 9.28 / 9.78 / 6.81 | AA                           |
| violet on-accent on accent / hover    | 7.35 / 9.35        | AA                           |
| green on-accent on accent / hover     | 8.80 / 10.22       | AA                           |
| amber on-accent on accent / hover     | 9.30 / 10.96       | AA                           |
| rose on-accent on accent / hover      | 7.90 / 9.52        | AA                           |

### More contrast

A future Settings › Interface › Contrast: More option changes three tokens and nothing else:

- **Light:** `--line` #b8bfca, `--line-strong` #7c8594 (3.72:1 on surface), `--muted` #464c58 (8.62:1).
- **Dark:** `--line` #3a3d44, `--line-strong` #6a6e78 (3.63:1), `--muted` #c6cad2 (11.29:1).

### Rules that follow from the numbers

- **Accepted exception.** The owner accepted success, warning and the green accent at 4.38 to 4.43:1 on white. Wherever they carry text, they are paired with an icon or set at weight 600, so meaning never rests on a thin coloured word. This applies to every status badge in checkship.
- **Control borders use `faint`, not `line-strong`.** `line-strong` is about 1.4:1 on surface, too weak to find a field; `faint` passes WCAG 1.4.11 (3:1) in both modes.
- **`faint` is never used for text** in either mode (3.18:1 on white). That includes placeholders, timestamps and disabled labels; use `muted`. Today's `::placeholder` uses `--co-text-3` in italics, which the migration replaces with `muted`, not italic.
- **Open items, not covered by the exception.** Success and warning on their soft fills (3.81 and 3.93), the light amber hover fill (3.67) and light amber as text (3.76 on surface) are below 4.5:1. Until the owner decides:
  - set soft-fill badges in `ink` with a coloured icon;
  - do not use the amber accent as a text or link colour in light mode.

---

## 5. Layout

Hierarchy is **Page → Section → Component**. A page has one title (its breadcrumb's last segment) and at most one primary action.

### Shell (built)

`src/lib/components/shell/`: `app-shell.svelte` (frame and breakpoints), `app-nav.svelte` (sidebar), `nav-link.svelte`, `theme-switch.svelte`, `account-menu.svelte`.

```
┌────────────────────────────────────────────────────────────────┐
│ ▣  ◯✓ checkship                                          ( A ) │ ← title bar, 44px, canvas
├──────────────┬─────────────────────────────────────────────────┤
│ Workspace    │  Applications                 [+ New application]│ ← page header, 40px below the panel top
│ ▦ Applications│  4 applications · 1 failed                      │ ← status line
│ ▤ Targets  ⚠ │  ┌─────────────────────────────────────────────┐ │
│ ▢ Registries │  │ row on sunken, 6px apart, no lines           │ │
│ ☑ Jobs     2 │  └─────────────────────────────────────────────┘ │
│ Recent       │                                                   │
│ ✓ welcome    │            workspace: surface, 6px corner         │
│              │            top left, centred column               │
│ ▤ Documentation ↗                                                │
│ Theme [▭ ☀ ☾]│                                                   │
│ checkship 2.4.2 · Source · Credits                               │
└──────────────┴─────────────────────────────────────────────────┘
```

- **Title bar.** 44px on `canvas`. Left: the sidebar toggle (1024px and up) or the menu button (below), then the logo at 22px. Right: the account avatar, a round `accent` circle with the email's initial, 36px (40px on phones). It opens the account menu: email, Profile, API key for CI, Sign out.
- **Sidebar.** On `canvas`, no edge line.
  - **Workspace** (from 24px under the title bar): Applications, Targets, Registries, Jobs; 40px items, 4px apart, Hugeicons at 20px. Targets shows a danger icon when a target is failing; Jobs shows the number of queued jobs.
  - **Recent:** the last five applications opened (stored in the browser, `#lib/preferences.svelte.ts`), each with its production state icon.
  - **Bottom:** Documentation (opens a new tab), the theme switch (System, Light, Dark), then the version with Source and Credits.
  - The current page is an `accent-soft` fill with `accent` text and weight 600.
- **Breakpoints.**
  - 1024px and up: sidebar 240px, collapsible to 56px from the title bar (remembered).
  - 768 to 1023px: sidebar 56px, icons with tooltips; the menu button opens the full sidebar as a sheet.
  - Under 768px: no sidebar; the menu button (44px) opens a 300px sheet with the logo at 28px.
- **Workspace.** The `surface` panel is flush to the right and bottom edges, with one 6px corner at the top left (top corners on phones). A 2px `accent` bar runs along the top of the window while SvelteKit navigates.
- **Page header** (`#lib/components/page-header.svelte`). As tall as its buttons (36px), 40px below the top of the panel: the title is the page's one `h1` (20/28), optionally after a breadcrumb; actions on the right, one primary, every labelled button with an 18px icon. A status line in `muted` follows, then 30px before the body.
- **Content width.** Every page fills the whole workspace (the floating `surface` panel next to the sidebar): no max width and no centring, at any screen size, with the same gap on every side (top, right, bottom, left): 20px on phones, 24px from 768px, 40px from 1024px. The page header is as tall as its buttons (36px), so the buttons sit exactly 40px below the top of the panel. Forms use two columns (`#lib/components/settings-section.svelte`, §11.6): the section title and one sentence on the left (260px), the fields on the right. Owner's call (2026-11-14): centred pages felt squeezed in the middle of wide screens, so the Profile › Content width setting was removed.
- **Lists.** Applications, Targets and Registries are rows, not cards: `sunken` fill, 6px corner, 6px apart, `hover` on hover; column labels above in caption `muted`. Below 768px a row stacks its cells with small labels.
- **Setting rows.** On `sunken`: the text column flexes, the control (switch, select, segmented control) never shrinks and stays on the right.
- **Form fields.** A label and its control stack vertically. Help text sits under the control in `muted`; a validation error replaces it in `danger`, with an icon (`#lib/components/field-message.svelte`).

### Auth layout (sign-in)

`src/routes/(auth)/+layout.svelte`. Two full-height columns from 1024px (`lg:grid-cols-2`, `min-h-svh`):

- **Left: the art panel** on `canvas`. A dithered illustration (`DitherImage`, source `static/auth/art.svg`: the ring-and-check mark with orbit rings) and the tagline, Heading 3 in `ink`, 48px from the left and bottom edges.
- **Right: the content** on `surface`, centred both ways. The sign-in form is one 360px column: logo 56px, title and description, fields, one full-width primary button.
- **No line between the columns.** The change from `canvas` to `surface` is the only boundary.
- **Below 1024px** the panel is hidden and the form is centred alone with 24px gutters. **Below 768px** inputs and the button are 44px tall with 16px text (touch targets; iOS does not zoom a 16px field).
- **The art is the one illustration besides the logo.** It is still flat: the dither turns the source's soft light into two solid tokens, ground pixels `canvas` and lit pixels `accent`, read from CSS so it follows light and dark. Ordered Bayer 8×8, 2px cells, threshold 0.45. It is decorative (`aria-hidden`) and bundled, so nothing loads from a third party.

### Spacing

The design rhythm is **6px**: 6, 12, 18, 24, 30 and 36px. _Why: it lines up with the 6px radius, so edges and gaps share one rhythm._

Tailwind's base unit stays 4px (`--spacing: 0.25rem`) because the shadcn-svelte components are sized on it. Layout code picks the classes that land on the 6px grid:

| Step | px  | Tailwind class          |
| ---- | --- | ----------------------- |
| 1    | 6   | `gap-1.5` / `p-1.5`     |
| 2    | 12  | `gap-3` / `p-3`         |
| 3    | 18  | `gap-4.5` / `p-4.5`     |
| 4    | 24  | `gap-6` / `p-6`         |
| 5    | 30  | `gap-7.5` / `p-7.5`     |
| 6    | 36  | `gap-9` / `p-9`         |

Component internals (button height, input padding) keep the shadcn defaults.

## 6. Shape

- **Every corner is 6px**: buttons, inputs, selects, cards, menus, popovers, dialogs, tooltips, toasts and the workspace surface.
- **Inner parts are 4px**: anything nested inside a 6px element, such as a menu item highlight, a tag inside a field or a code chip.
- **Exceptions are controls whose shape carries meaning.** These are the radio dot, the switch track and thumb, the avatar and the status dot, which are all round.
- **No other pills.** Badges are 4px-cornered, not `rounded-full`. Change this in `badge.svelte` after adding it. _Why: one radius makes the interface feel built from one material, and 6px is soft without looking rounded._

## 7. Type

- **Inter**, bundled with `@fontsource-variable/inter` so the dashboard works offline and loads nothing from a third party. Today only `@fontsource/inter/latin-600.css` is loaded, for the logo wordmark, and the body uses the system stack. The migration makes Inter the body font.
- **16px base**, with everything in rem.
- **Scale**, carried over from today's `--ty-*` tokens:

  | Role      | Size / line height | Weight | Tailwind                          |
  | --------- | ------------------ | ------ | --------------------------------- |
  | Heading 1 | 30 / 36            | 600    | `text-3xl font-semibold`          |
  | Heading 2 | 20 / 28            | 600    | `text-xl font-semibold`           |
  | Heading 3 | 18 / 28            | 600    | `text-lg font-semibold`           |
  | Body      | 16 / 24            | 400    | `text-base`                       |
  | Caption   | 12 / 16            | 400    | `text-xs`                         |
  | Mono      | 12 / 16            | 400    | `font-mono text-xs`               |

- **Headings** are semibold (600) with tight tracking (`tracking-tight`).
- **Descriptions** under headings and form fields are `muted`, at regular weight.
- **Mono** is the system mono stack (`ui-monospace, SFMono-Regular, Menlo, …`). It is used for logs, compose files, environment variables, URLs, API keys, image names, job payloads and commit hashes.
- **Numbers** that update in place, such as deployment numbers, durations and counts, use `tabular-nums`.

## 8. Icons

- **Hugeicons, stroke rounded, 1.5 stroke** (the owner's choice over Lucide, Tabler and Phosphor, 2026-11-14), from `@hugeicons/core-free-icons` (MIT). shadcn-svelte's `components.json` is set to `"iconLibrary": "hugeicons"`.
- **One component draws them:** `#lib/components/icon.svelte` renders the icon data as plain SVG, so icons paint with the page (the package's own Svelte component draws after mount). Import per icon so the build tree-shakes:

  ```svelte
  import Icon from '#lib/components/icon.svelte';
  import Add01Icon from '@hugeicons/core-free-icons/Add01Icon';
  <Icon icon={Add01Icon} class="size-4" />
  ```

- **Sizes:** 18px in buttons, 20px in the sidebar and title bar, 14px in badges.
- **Every labelled button leads with an icon** that names its action (New: `Add01`, Save: `Tick02`, Promote: `Rocket01`, Redeploy and Reconfigure: `Refresh`, Copy: `Copy01`, Delete: `Delete02`). The icon is decorative: the label carries the meaning.
- **Colour.** Icons inherit `currentColor`.
- **Accessibility.** A decorative icon is `aria-hidden` (the default). An icon-only button has an `aria-label` and a `Tooltip`.
- **Nav icons:** Applications `DashboardSquare01`, Targets `ServerStack01`, Registries `Package`, Jobs `TaskDone01`, Documentation `Book02`, theme `Computer` / `Sun03` / `Moon02`, sidebar toggle `SidebarLeft`, menu `Menu01`.
- **Status icons:** `CheckmarkCircle02` succeeded or ready, `CancelCircle` failed, `Loading03` (spinning) running or configuring, `Clock01` pending, `Delete02` deletion requested, `Alert02` warnings.
- **Old icons** (`src/assets/icons/*.svelte`) stay until the last old page that uses them is rebuilt.

## 9. Motion

- **Allowed:**
  - transitions on state change (150ms);
  - the navigation progress bar;
  - the spinner while work is pending;
  - Bits UI's open and close animations for menus, popovers, dialogs and the sheet (from `tw-animate-css`).
- **Nothing loops** except a spinner while work is pending. The logo never animates, and the sign-in art is drawn once per size or theme change, never animated.
- **Reduced motion.** When the OS asks for it, every transition and animation drops to near-instant:

  ```css
  @media (prefers-reduced-motion: reduce) {
    *, ::before, ::after { animation-duration: 1ms !important; transition-duration: 1ms !important; }
  }
  ```

_Why: motion should confirm that something happened. Looping motion competes with the content and costs battery._

## 10. Logo

- **The mark is a ring-and-check, stroke only.** There is no tile, fill, gradient or shadow. It is drawn at stroke 2 in a 24 × 24 viewBox with round caps and joins.
- **Stroke colour** is the brand colour: `#2f5bea` in light and `#7f95ff` in dark. It never follows the accent tone.
- **Wordmark.** "checkship" in Inter 600, lower case, in `ink`, with -0.02em tracking.
- **Spacing.** The mark sits left of the wordmark with a gap of 0.4× the mark size.
- **Sizes:**

  | Place                         | Size  |
  | ----------------------------- | ----- |
  | Title bar (top bar on mobile) | 22px  |
  | Sidebar header                | 28px  |
  | Sign-in screen                | 56px  |
  | Settings › About              | 72px  |

- **Favicon.** The favicon (`static/favicon.svg`) is the mark alone and adapts to light and dark.
- **App icon.** The same mark at stroke 2.25 on a 512px transparent canvas. `scripts/build-icons.mjs` renders it to `assets/brand/icon.png` and `icon.ico`; regenerate them with `npm run build:icons` at the repo root.
- **Sources.** The source SVGs are `assets/brand/logo-mark.svg` and `logo.svg`. In the dashboard the logo is `src/assets/logo.svelte` (props `size`, `markOnly`, `class`).

---

## 11. The app today: pages, components, patterns

### 11.1 Pages

The dashboard is a SvelteKit SPA. `ssr = false`, `prerender = false` and `trailingSlash = 'always'` are set in `src/routes/+layout.ts`. It is built by `adapter-static` with `fallback.html` and embedded in the Go binary.

The `(main)` group's `+layout.ts` checks the session (`GET /api/v1/profile`) and the health check (`GET /api/v1/healthcheck`) before any page renders. If the session check fails, it redirects to `/signin?redirectTo=…`.

| # | URL | File (under `src/routes`) | What the page is | Loads / polls | Actions (API) |
|---|-----|---------------------------|------------------|---------------|---------------|
| 1 | `/signin/` | `(auth)/+layout.svelte` + `(auth)/signin/+page.svelte` | Sign in (**migrated**): art panel + centred form, see §5 Auth layout. `invalid_email_or_password` shows once as an alert; other codes under their field | – | `POST /sessions` |
| 2 | `/` | `(main)/+page.svelte` | **Applications.** Grid of `AppCard`, or a blank slate | `GET /apps`; polls every 5s | link to New application |
| 3 | `/apps/new/` | `(main)/apps/new/+page.svelte` + `app-form.svelte` | **New application.** Name, the URLs it will get, version control (URL, token), production and staging targets and env vars | `GET /targets?active_only=true` | `POST /apps` |
| 4 | `/apps/[id]/` | `(main)/apps/[id]/+page.svelte` | **Application.** Production and staging `EnvironmentCard`s with the latest deployments, or a blank slate. Shows a cleanup notice if deletion was requested | `GET /apps/{id}` (404 → section `+error`); polls every 5s | Edit, New deployment |
| 5 | `/apps/[id]/edit/` | `(main)/apps/[id]/edit/+page.svelte` + `app-form.svelte` | **Application settings.** The same form, plus Delete | – | `PATCH /apps/{id}`, `DELETE /apps/{id}` (asks to confirm) |
| 6 | `/apps/[id]/deployments/new/` | `(main)/apps/[id]/deployments/new/+page.svelte` | **New deployment.** Environment, then payload kind: Raw compose file, Archive (.tar.gz) or Git (branch / commit). The equivalent cURL command can be shown in a `Console` | – | `POST /apps/{id}/deployments` |
| 7 | `/apps/[id]/deployments/[number]/` | `(main)/apps/[id]/deployments/[number]/+page.svelte` | **Deployment #n.** `DeploymentCard` (state, source, branch or commit, services and their URLs, timings) and the logs `Console`. Marks the deployment stale when a newer one exists | `GET …/deployments/{n}` and `…/logs`; polls every 2.5s until finished and the target is not configuring | Promote, Redeploy (both ask to confirm) |
| 8 | `/targets/` | `(main)/targets/+page.svelte` | **Targets.** Grid of `TargetCard` (state, URL, provider) | `GET /targets`; polls every 5s | link to New target |
| 9 | `/targets/new/` | `(main)/targets/new/+page.svelte` + `target-form.svelte` | **New target.** Name, "expose services automatically" with its URL, the Docker provider (local or remote: host, user, port, SSH key) | – | `POST /targets` |
| 10 | `/targets/[id]/edit/` | `(main)/targets/[id]/edit/+page.svelte` + `target-form.svelte` | **Target settings.** State panel (configuring, ready or failed with the error), the form, Reconfigure and Delete | – | `PATCH /targets/{id}`, `POST /targets/{id}/reconfigure`, `DELETE /targets/{id}` |
| 11 | `/registries/` | `(main)/registries/+page.svelte` | **Registries.** Grid of `RegistryCard` | `GET /registries`; polls every 5s | link to New registry |
| 12 | `/registries/new/` | `(main)/registries/new/+page.svelte` + `registry-form.svelte` | **New registry.** Name, URL, credentials | – | `POST /registries` |
| 13 | `/registries/[id]/edit/` | `(main)/registries/[id]/edit/+page.svelte` + `registry-form.svelte` | **Registry settings.** Form and Delete | – | `PATCH /registries/{id}`, `DELETE /registries/{id}` |
| 14 | `/jobs/` | `(main)/jobs/+page.svelte` + `cancel-button.svelte` | **Jobs.** Paginated `DataTable`: resource, status, dates; expands to group, payload, policy, error | `GET /jobs?page=n`; polls every 5s | `DELETE /jobs/{id}` (cancel) |
| 15 | `/profile/` | `(main)/profile/+page.svelte` | **Profile.** Email and password, interface language, integration (API key, regenerate) | `GET /profile` | `PATCH /profile`, `PUT /profile/key` |
| – | any | `+error.svelte`; section `+error.svelte` in `apps/`, `targets/`, `registries/` and `deployments/[number]/` | Not found and unexpected error | – | back links |

All API paths are under `/api/v1`. Shell pieces live in `(main)/`:

- `topbar.svelte`: logo, nav and the account box;
- `account.svelte`: email, Profile and Sign out;
- `bottom-bar.svelte`: version, Source, Docs and Credits;
- `progress-bar.svelte`.

**Planned pages.** Settings › About is where the 72px logo goes, together with the version, the GPL-3.0 notice and the credits; Profile will move under Settings. Nothing about these pages is built yet.

### 11.2 Components and their shadcn-svelte replacements

Everything is in `src/components/` (alias `$components`). Each component maps to what replaces it after the migration (section 12):

| Today | Purpose | Replace with |
|-------|---------|--------------|
| `button-base.svelte`, `button.svelte` | Button or link-button; variants primary, outlined and danger; `loading` spinner; label from a translation key | `Button` (variants `default`, `outline`, `destructive`, `ghost`, `link`) + `Spinner` |
| `link.svelte` | Styled anchor | `Button variant="link"` or a plain `<a>` with `text-accent` |
| `text-input.svelte`, `text-area.svelte`, `input-adorner.svelte`, `input-help.svelte` | Labelled input or textarea with help text and remote error | `Field` + `Label` + `Input` / `Textarea` (+ `InputGroup` for adornments) |
| `dropdown.svelte` | Native `<select>` | `Select` (Bits UI) |
| `checkbox.svelte` | Labelled checkbox that reveals a section | `Checkbox` or `Switch` + `Label` |
| `input-file.svelte` | File input for .tar.gz | `Input type="file"` inside `Field` |
| `form.svelte`, `form-section.svelte`, `form-errors.svelte` | `<form>` + fieldset disabled while submitting; titled section; error list | Keep `form.svelte` (it wraps `submitter`), re-styled; `FieldSet` / `FieldLegend` for sections; `Alert variant="destructive"` for errors |
| `form-env-vars.svelte` | Per-service env var editors | `Field` + `Textarea` (mono) inside `Collapsible` per service |
| `card.svelte`, `cards-grid.svelte` | Entity card with optional footer; responsive grid | **No cards:** rows on `sunken` (§5, Lists). Done for applications, targets, registries |
| `app-card`, `target-card`, `registry-card`, `environment-card`, `deployment-card` | Domain cards | Same names, rebuilt on `Card` |
| `deployment-pill`, `entrypoint-pill` | Pill links (deployment number + state; service URL) | `Badge` (4px corner) with a status icon |
| `status-indicator.svelte` | Coloured dot for success, failed, running, pending | Status dot + `Badge`; rules in 11.4 |
| `panel.svelte` | Info, warning, danger or success callout; can be collapsible or inline | `Alert` (default, destructive + soft variants); `Collapsible` when collapsible |
| `cleanup-notice.svelte` | "Deletion requested" notice | `Alert` (danger-soft) or `Badge` |
| `console.svelte` | Monospace log or command viewer with copy | `ScrollArea` + `<pre>` on `sunken`, copy `Button` |
| `data-table.svelte` | Responsive table; rows expand with a CSS checkbox; cells become label/value pairs on mobile | `Table` + `Collapsible` rows |
| `pagination.svelte` | Previous and next pager | `Pagination` |
| `deployments-list.svelte` | List of deployments | `Item` list or `Table` |
| `breadcrumb.svelte`, `page-title.svelte` | Page header with actions; `<title>` | `Breadcrumb` in the 48px header; keep `page-title` (`<svelte:head>`) |
| `blank-slate.svelte` | Empty message | `Empty` (11.5) |
| `display.svelte` | Label/value pair | `Item` or a `dl` |
| `prose.svelte` | Rich text styling | Tailwind typography utilities, or keep |
| `stack.svelte` | Flex helper with gap tokens | Tailwind `flex` / `grid` + `gap-*` (removed) |
| `loader.svelte` | Spinner | `Spinner` |
| `service-info.svelte` | A service's image and URLs | `Item` |
| `topbar`, `account`, `bottom-bar` (routes) | Shell | `Sidebar`, `DropdownMenu`, `Avatar`, `Sheet` (mobile) |
| `progress-bar` (route) | Navigation bar | Keep; restyle as a 2px `accent` bar |

### 11.3 Shared logic (`src/lib/`)

- **`fetcher/`.** `FetchService` with a cache (`cache.ts`) and an invalidation set (`set.ts`).
  - `get` fetches once.
  - `query` returns Svelte stores `{ data, loading, error }` and can poll with `refreshInterval`.
  - `post`, `patch`, `put` and `delete` invalidate their URL and any extra `invalidate` URLs.
  - `reset()` clears everything at sign-in.
- **`resources/*.ts`.** One file per API resource: `apps`, `deployments`, `targets`, `registries`, `jobs`, `sessions`, `users`, `healthcheck`. Each holds the types (mirroring the Go DTOs, with snake_case fields), an interface, a `Remote…Service` class taking the fetcher and options, and a default instance.
- **`form.ts`.** `submitter(fn, { confirmation })` gives `{ submit, loading, errors }` stores and maps `BadRequestError` validation fields to `errors[field]` (`__global` otherwise). `buildFormData` builds multipart bodies, and `messageFromAttributes` builds the client-side hint.
- **`error.ts`.** `HttpError.from(response)` maps statuses: 400 → `BadRequestError` (`validation_failed` → fields), 401 → `UnauthorizedError`, 500 → `UnexpectedError`.
- **`localization/`.** Typed keys in `en.ts` and `fr.ts`; there are 187 entries.
  - `l.translate(key, args?)` is checked against the key type, and translation functions take arguments.
  - Changing the locale reloads the page; translations are deliberately not reactive.
  - There are also `l.date`, `l.datetime` and `l.duration`.
- **`auth.ts`.** Login and logout with redirects (`redirectTo`), plus `checkSession`.
- **`path.ts`.** Every route as a constant or function (`routes.deployment(id, n)`). Pages never hand-write URLs.
- **`config.ts`.** `POLLING_INTERVAL_MS = 5000` and `RUNNING_DEPLOYMENT_POLLING_INTERVAL_MS = 2500`.
- **Small helpers.** `curl.ts` builds the cURL command for a deployment, and `select.ts` is a typed discriminated-union switch. `aria.ts` holds `controls`, an action for `aria-expanded` and `aria-controls`; `actions.ts` holds `useAction` for passing an action into a component.
- **Tests** sit next to their source as `*.test.ts` (vitest). They cover `curl`, `select`, `localization`, and the fetcher's `cache`, `data` and `set`.

### 11.4 States and their colours

| Domain | State | Token | Icon | Words |
|--------|-------|-------|------|-------|
| Deployment | pending | `muted` | `clock` | Pending |
| Deployment | running | `warning` | `loader-circle` (spins) | Running |
| Deployment | succeeded | `success` | `circle-check` | Succeeded |
| Deployment | failed | `danger` | `circle-x` | Failed + the error code |
| Target | configuring | `warning` | `loader-circle` | Configuring |
| Target | ready | `success` | `circle-check` | Ready |
| Target | failed | `danger` | `circle-x` | Failed + the error (e.g. the Docker daemon is unreachable) |
| Job | pending / running / failed | `muted` / `warning` / `danger` | as above | + retry count |
| App, target or registry | cleanup requested | `danger` | `trash-2` | Deletion requested on … |
| Environment | production / staging | `ink` badge | – | the name; never coloured |

- **Badge styling.** A status badge is the soft fill (`success-soft` and so on) with **`ink` text** and the **coloured icon** (section 4 rules). The dot alone is never the only signal.
- **Removed colour.** Today's emerald `--co-primary-*` is retired; the primary action is `accent` blue.

### 11.5 Patterns

- **Empty states.** Every empty state has exactly four parts:
  1. a line icon or SVG in `faint`, with one node in `accent`;
  2. a title;
  3. one sentence saying what will appear here;
  4. one action, the most likely next step (New application, New target and so on).

  Today's blank slates use `{@html}` translations with inline links. In the migration they become the `Empty` component with a real `Button`.
- **Spinners in buttons.** Every button that starts async work has a loading state. On the shadcn `Button` it is the `loading` prop (`<Button loading={$submitting}>`):
  - the spinner is mounted only while pending;
  - the label stays, so the width does not change;
  - the button stays focusable (`aria-busy`, never `disabled`) and ignores clicks, which also blocks a second submit;
  - the spinner is `aria-hidden`.

  The old `button-base` (`loading` prop) keeps its label next to the spinner too.
- **Forms.** Validation errors from the API (`validation_failed` → field codes) show under each field. Anything else shows in an `Alert` at the top of the form. While submitting, the fieldset is disabled and the submit button shows its spinner. Required fields are marked in the label, not by colour alone.
- **Confirmations.**
  - **Promote and Redeploy** use an `AlertDialog` that names the deployment number and environment; this replaces `window.confirm`.
  - **Deleting** an application, target or registry confirms in an `AlertDialog` that names the thing and what happens: its resources are cleaned up on the target. Deleting a target also asks for its name to be typed.
  - **Cancelling a job** confirms in an `AlertDialog`.
- **Toasts** (`svelte-sonner` via the shadcn `Sonner` component). Toasts are for results whose own UI has closed: copied to clipboard, API key regenerated, job cancelled, target reconfiguring.
  - They are bottom right, with at most three at once.
  - They use the `surface` fill, a `line` border and a 6px corner.
  - Errors stay 8 seconds; everything else stays 4.
  - Never put a decision in a toast.
- **Polling.** Lists poll every 5s and a running deployment every 2.5s; polling stops once the deployment has finished. A refreshing list never flashes a spinner; old data stays until new data arrives.
- **Logs and code.** These are monospace on `sunken`, in a `ScrollArea` capped at 70vh and pinned to the bottom while the deployment runs. A copy button copies everything.
- **Long values.** URLs, image names and paths that don't fit are truncated in the middle with an ellipsis, keeping the end (the host, the tag). They have a copy button where the value is meant to be reused, such as the API key, the cURL command and service URLs.
- **Focus.** Focus is drawn **inside** the control, never as an outer ring or with an offset. A bordered control (input, select, checkbox, outline button) turns its border `accent` and adds a 1px inset ring, so it reads as a 2px accent edge (`focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-inset focus-visible:ring-ring`). A filled button gets a 2px inset ring in its foreground colour (`on-accent` on the primary button). Plain elements fall back to a 2px `accent` outline at a -2px offset. Focus is never removed without this replacement. The old pages' 3px emerald outline goes as they are migrated.

### 11.6 Drawers, drop zones and highlighted code

Added 2026-11-14 after the owner's review of the "application pages · v2 drawers" canvas.

- **Drawer, dialog or page.**
  - **Drawer** (`#lib/components/drawer.svelte`, the Bits UI `Sheet` on the right) for short tasks where the page behind still matters: New deployment and the deployment preview on the application page, the environment variables on the settings page, a job's details, New registry and a registry's settings.
  - **Dialog** (`ConfirmDialog`) for destructive confirmation only: delete, cancel a job. Never a drawer.
  - **Page** for long forms and anything that needs a link: New application, targets, the full deployment. Old links keep working: `/apps/:id/deployments/new`, `/registries/new` and `/registries/:id/edit` render the same form as a page.
- **Drawer anatomy.**
  - 560px for forms, 680px for previews; never wider than half the window. Full screen below `md`, with 44px buttons that share the footer.
  - From `md` up it floats 8px from the window edges with a 10px corner. `surface` fill, no border, a soft shadow; the scrim is `rgb(5 5 7 / 0.62)`.
  - Header: title, one line of description, a close button. The body scrolls. Footer: actions on the right, primary last; a destructive action sits on the left. A shadow appears above the footer only while there is more to scroll.
  - Esc and the scrim close it; focus goes into it and returns to what opened it. Forms inside a drawer have an `id`, and the footer's submit button uses `form="…"`.
  - The row that opened a drawer keeps the `raised` fill while it is open.
- **Drop zones** (`#lib/components/drop-zone.svelte`). A `<button>` that also takes a dropped file, so it is focusable and opens the file picker.
  - Its label names both ways in ("Add a compose.yml: drop it here or choose a file"), and it says what it accepts before anything is dropped.
  - While a file is dragged over: `accent-soft` fill, a 2px inset accent ring, "Release to add the file".
  - Compact (a row) for the compose file and `.env` files, which fill an editor; the block form for the archive. A wrong file is refused in its file row, which turns `danger-soft` and says why.
- **Highlighted code** (`#lib/highlight.ts`, Shiki 4.6). The core, the JavaScript regex engine (no WASM) and four grammars (yaml, json, shellscript, dotenv) load on first use. The theme is Shiki's css-variables theme mapped to palette tokens in `layout.css` (keys `accent`, strings `success`, constants `warning`, comments `faint`, punctuation `muted`), so it follows light and dark with no new colour. Logs get a small rule instead of a grammar: faint timestamps, `danger` for error lines.
  - Read-only code (`code-block.svelte`) sits on `canvas` with an optional Copy button.
  - Editable code (`code-editor.svelte`) is a transparent textarea over the highlighted text in the same grid cell, so it grows with its content. It looks like any input: `sunken`, `faint` border, inset focus.
- **Settings in two columns.** The section title and one sentence on the left (260px), the fields on the right. Each environment is one row: name with a summary, the target select, and a Variables button that opens the drawer. On an existing application the drawer saves that environment's variables at once (only that environment is sent); on a new one it hands them back to the form.

### 11.7 Application details

Added 2026-11-14 after research into Vercel, Render, Railway and Coolify (the "Application details v3" canvas row). Only patterns checkship has the API for were kept.

- **Header** (`routes/(main)/apps/[id]/app-workspace.svelte`): the name, a status line (what each environment serves: live, deploying, failed or not deployed, with its number; the repository; who created it), then Visit (the production URL), Settings and a split **Deploy** button.
  - Deploy is one click: the latest commit of the branch production last deployed (Coolify: "a one-click button … the adjacent dropdown for secondary actions"; Render's Manual Deploy). With no Git repository, or no branch yet, it opens the New deployment drawer instead.
  - The arrow opens the other ways: deploy staging, a compose file, an archive, redeploy production. Each queued deployment opens its preview drawer.
- **Tabs** (`app-tabs.svelte`): Overview, Deployments (with the total), Settings. They are links, so each has its own address; the current one has the `raised` fill. Variables stay in Settings.
- **Overview.** Staging and production as a pipeline: two cards with **Promote** between them, which names the deployment and says why it is offered (staging runs a newer commit), or why it is not (still running, failed, nothing on staging). Each card leads with the live URL, then a Live / Deploying / Failed marker, the state badge, source with branch and short commit (or the error code), and Logs and Redeploy. A running deployment adds a banner that says production keeps serving its current one. Below: the services of both environments (URL and image per environment) and the five most recent deployments.
- **Deployments tab** (`deployments/+page.svelte`): environment and status as segmented controls and a search on number, branch or commit, all filtered on the server (`status`, `search` and `per_page` query parameters, 20 per page). The deployment each environment serves is marked Live. Each row opens its preview; its menu (Railway) has View logs, Promote to production (successful staging rows), Redeploy and Copy link. Promote and Redeploy confirm in a dialog.
- **Not added, on purpose:** restart and stop, runtime logs, metrics, a terminal, rollback without rebuilding. checkship has no endpoint for them yet; they can join the Deploy menu and the tabs when it does.

### 11.8 Targets and registries

Added 2026-11-14, researched (Coolify servers and proxy, Dokploy remote servers and registries, Docker, GitHub, GitLab and Quay docs) and checked line by line against the Go backend.

- **Targets list:** one row per Docker engine with where it is (`This machine · Docker engine` or `user@host:port`, with the server's defaults docker and 22 filled in), how it exposes services, its state and how many applications use it. Ready shows when it was configured: `last_ready_version` is a time, not a version. Failed shows its error and a Reconfigure button on the row. A target being deleted shows Deleting. Rows open the target.
- **Target page** (`routes/(main)/targets/[id]`): tabs Overview and Settings. The overview has the state (failed: the error in its own block, three things to check, Reconfigure), Connection and Exposure as fact cards, and the applications that deploy to it.
- **Form, following `infra/provider/docker/data.go` and `domain/target.go`:**
  - One target per Docker engine: "This machine" is offered only while no local target exists.
  - The host identifies a remote target and can't change (`target_provider_update_not_permitted`): read-only in Settings.
  - User defaults to docker, port to 22. The private key is optional (else the SSH config of the user running checkship); it can be dropped as a file or pasted.
  - A URL means automatic exposure: root only, unique among targets; the form previews the app URLs and the wildcard DNS record.
  - Changing the URL, exposure, user, port or key reconfigures the target; renaming doesn't.
- **Delete:** refused while environments use the target (`target_in_use`): the dialog lists them, each linking to its settings. Refused while configuring. A target never ready is deleted without cleanup.
- **Secrets are masked, not encrypted.** `storage.SecretString` replaces every character with `*` in API responses; nothing is encrypted at rest. Copy says "never shown again", never "encrypted". The form never fills a masked value back in.
- **Registries:** a new registry starts from a preset with its official mark (`#lib/registry-logos.ts`: Simple Icons 16.37.0, CC0, traced from each brand's kit; Quay's mark from quay/quay). Marks keep their brand colours (Docker blue, GitLab orange, Quay blues); GitHub's uses the text colour, as GitHub allows black, white or grey. Each preset fills the name and URL and names the token: Docker Hub a read-only personal access token, GitHub a classic token with `read:packages`, GitLab a deploy token with `read_registry`, Quay a robot account (`namespace+robot`). Saved registries are recognised from their URL. Server rules: name required, URL required and unique, username and password together; turning sign-in off removes the login. Docker uses each login, keyed by the URL, for every deployment on every target.

### 11.9 Jobs and profile

Added 2026-11-14, researched (Oban Web, SolidQueueWeb, Bull Board; SaaS UI patterns for API keys, profile and settings pages; Zuplo's API key practices) and checked against `pkg/bus/sqlite/store.go` and `internal/auth/app`.

- **Jobs list** (`routes/(main)/jobs`): the live queue, 10 per page, oldest first (the server's order, said on the page). The header shows the count, a Live dot (polls every 5 s) and that failed jobs retry on their own every 15 seconds. Pause updates stops polling; it also pauses while a job's drawer is open and while the tab is hidden.
- **Row states, from the server's fields** (`#lib/job.ts`): Running (`retrieved`), Failed, retrying (an error; "next try in N s" from `not_before`), Waiting (a running job of its group, or an earlier job on its resource for jobs that wait for it, is on the same page; "after welcome #5"), Pending. Each row is named for what it acts on: a deployment as "app #N" with its environment (read from the group, `deployment.deployment.deploy.<app>-<env>-<id>`), cleanups by app, target jobs by target. The third column shows the error, else the resource id.
- **State filter:** `GET /api/v1/jobs?status=running|failed|pending` (`bus.GetJobsFilters.Status`); pending means not retrieved and never failed.
- **Actions follow the policy:** Cancel only on cancellable jobs (cleanups), behind a confirmation naming the job. No Retry (automatic) and no bulk actions (the API cancels one job at a time). Deployments can't be cancelled.
- **Job drawer:** state, last error with what happens next, For (links to the deployment, app or target), Resource, Group (one job of a group runs at a time), Queued, Not before, the policy in plain words, and the JSON payload with Copy. Footer: Open deployment/application/target, and Cancel job or Close.
- **Profile** (`routes/(main)/profile`): who you are (avatar, email, account created) with Sign out, then sections that save on their own and confirm it: Account (email, valid and unique, `email_already_taken`), Password (typed twice and compared in the page; the API only needs it non-empty and doesn't ask for the current one), Interface (language, applied at once with a reload).
- **API key:** the server returns it in full, so the page masks it (first and last four) with Reveal and Copy. A highlighted cURL example shows how CI uses it. Regenerate sits in a danger-tinted block behind a confirmation: the old key stops working at once.
- **Not added, no API:** sessions and "sign out everywhere", two-factor, account deletion, key last-used date, several keys.

---

## 12. UI library: research and choice

### What "Radix" means here

checkship is Svelte, and Radix Primitives is React-only, so Radix itself can't be used.

- **The Svelte equivalent is [Bits UI](https://bits-ui.com).** It provides headless, accessible primitives with the same compound-part model as Radix (`Dialog.Root`, `Dialog.Trigger`, `Dialog.Content`, …), plus focus management, typeahead, roving focus and portals.
- **[shadcn-svelte](https://shadcn-svelte.com) sits on top of it.** It is the Svelte port of shadcn/ui, which is itself Radix + Tailwind. It copies styled Bits UI components into the repo, so the code is ours to edit to the rules above.

### Options compared

| Library | Version (npm, 2026-11-14) | Model | Verdict |
|---------|---------------------------|-------|---------|
| **Bits UI** | 2.21.0 | Headless components, Svelte 5 runes and snippets, Radix-style parts | **Chosen primitive layer** |
| **shadcn-svelte** | 1.8.1 (CLI) | Copy-in styled components over Bits UI + Tailwind v4; the CLI writes code you own | **Chosen component layer** |
| Melt UI (`melt`) | 0.44.0 (pre-1.0) | Low-level builders; the old `@melt-ui/svelte` 0.86 is the Svelte 4 store API | More work per component and not 1.0 yet; Bits UI already covers our needs |
| Ark UI (`@ark-ui/svelte`) | 5.25.1 | Zag.js state machines shared with React and Vue | Good, but its value is cross-framework and we are Svelte only. No shadcn-style copy-in set |
| Skeleton, Flowbite-Svelte, DaisyUI | – | Pre-styled kits with their own look | Fight the palette and the 6px-only shape rules |

**Why shadcn-svelte on Bits UI:**

1. **Accessibility out of the box.** We get the keyboard and screen-reader behaviour of Radix (focus trap, `aria-*`, Escape and outside-click) without writing it. Today's checkbox-hack menu and CSS-checkbox table rows have none of it.
2. **We own the code.** The components land in `src/lib/components/ui/` as plain `.svelte` files, so the 6px radius, flat fills and badge shape are edited once, in our repo.
3. **Tokens map cleanly.** shadcn's theme is CSS variables plus Tailwind `@theme inline`, and our palette is CSS variables (section 13.3).
4. **It is the most-used Svelte 5 component set, and it is actively maintained.** The latest CLI release was on 2026-11-11, and SvelteKit 3 support landed in October 2026 (issues #2849, #2953 and #2915 are closed).
5. **The shadcn ecosystem pieces we need are Svelte 5 native:** `svelte-sonner` (toasts), `mode-watcher` (theme) and Hugeicons (icons; chosen over Lucide on 2026-11-14).

### Components to add

Every component below has a home in a page or pattern above:

```
button  input  textarea  label  field  input-group  select  checkbox  switch
card  badge  alert  alert-dialog  dialog  sheet  dropdown-menu  tooltip
sidebar  breadcrumb  table  pagination  collapsible  scroll-area  separator
skeleton  spinner  empty  item  avatar  sonner  kbd
```

These were deliberately left out:

- **`form` (formsnap + sveltekit-superforms + zod).** Validation lives in the Go API and arrives as `validation_failed` field codes, and the app is an SPA with no SvelteKit form actions. `field` (no dependencies) plus our `submitter` covers it with zero new runtime dependencies.
- **`data-table` (TanStack Table).** The Jobs table is one paginated list, and `table` is enough.
- **`chart` (LayerChart), `resizable` (paneforge), `drawer` (vaul-svelte, whose 1.0 is still on `next`), `calendar`, `command`.** No page needs them. Add one when a page does, not before.

---

## 13. Dependencies and the migration

### 13.1 Where we are

`cmd/serve/front/package.json` today:

| Package | Today | Problem |
|---------|-------|---------|
| svelte | 3.59.2 | Bits UI and shadcn-svelte need Svelte 5 |
| @sveltejs/kit | 1.20.2 | Kit 3 is current (3.2.0); Kit 2's last line is 2.70.4 |
| @sveltejs/adapter-static | 2.0.2 | Needs 4.x for Kit 3 |
| vite | 4.3.9 | Kit 3 needs Vite ≥ 8.0.12 |
| typescript | 4.9.5 | Kit 3 needs TS 6 |
| vitest | 0.29.8 | Current is 5.1 (Vite 8) |
| svelte-check | 3.6.1 | Current is 4.7 |
| eslint 8 + eslint-plugin-svelte3 + @typescript-eslint 5 | – | `eslint-plugin-svelte3` is dead; move to ESLint 10 flat config |
| prettier 2 + prettier-plugin-svelte 2 | – | Prettier 3 and plugin 4 |
| svelte-preprocess + svelte-preprocess-cssmodules | – | `<style module>` + `bind()`; replaced by Tailwind classes |
| the-new-css-reset | – | Replaced by Tailwind preflight |
| @fontsource/inter 5.2.6 (latin-600) | – | Replace with `@fontsource-variable/inter` |

The toolchain outside `package.json` also changes:

- **Dockerfile:** `FROM node:18-alpine`.
- **CI:** `.github/workflows/ci.yml` uses `node-version: 20`.
- **Requirement:** SvelteKit 3 needs **Node ≥ 22.17**. Move both to Node 24 (LTS).

### 13.2 Target versions (checked on npm 2026-11-14)

**Framework and tooling (devDependencies):**

| Package | Version | Note |
|---------|---------|------|
| svelte | 5.57.3 | runes, snippets |
| @sveltejs/kit | 3.2.0 | config moves to `vite.config.ts`; `$lib` → `#lib` |
| @sveltejs/vite-plugin-svelte | 7.3.1 | peer of Kit 3 |
| @sveltejs/adapter-static | 4.0.1 | same `fallback: 'fallback.html'` |
| vite | 8.3.8 | Rolldown-based |
| typescript | **6.0.3** | **not 7.x.** Kit 3 needs `^6`, svelte-check accepts `^5 \|\| ^6`, typescript-eslint needs `<6.1` |
| svelte-check | 4.7.8 | |
| vitest | 5.1.0 | |
| tailwindcss + @tailwindcss/vite | 4.3.5 | CSS-first config |
| tw-animate-css | 1.4.0 | added by shadcn init |
| eslint | 10.15.0 | flat config `eslint.config.js` |
| eslint-plugin-svelte | 3.24.1 | |
| typescript-eslint | 8.72.2 | |
| prettier | 3.10.1 | |
| prettier-plugin-svelte | 4.1.1 | |
| prettier-plugin-tailwindcss | 0.8.1 | class sorting |

**UI (dependencies or devDependencies; everything is bundled into the SPA either way):**

| Package | Version | Why |
|---------|---------|-----|
| bits-ui | 2.21.0 | primitives (installed by shadcn) |
| @internationalized/date | 3.12.4 | required peer of bits-ui |
| shadcn-svelte | 1.8.1 | CLI only, run with `npx`, not installed |
| cn · tailwind-variants | 0.5.2 · 3.3.1 | `cn()` class merging (shadcn's own package, replaces clsx + tailwind-merge) and component variants; both installed by shadcn |
| @hugeicons/core-free-icons | 4.4.0 | icons (replaced @lucide/svelte, 2026-11-14) |
| svelte-sonner | 1.2.3 | toasts |
| mode-watcher | 1.1.0 | light, dark and system theme |
| @fontsource-variable/inter | 5.3.0 | body and wordmark font |

**Removed:** svelte-preprocess, svelte-preprocess-cssmodules, the-new-css-reset, @fontsource/inter, eslint-plugin-svelte3, @typescript-eslint/eslint-plugin, @typescript-eslint/parser, eslint-config-prettier (re-add only if the flat config needs it).

**Freshness.** bits-ui 2.21.0, shadcn-svelte 1.8.1 and @sveltejs/kit 3.2.0 are each only a few days old on 2026-11-14. If a step breaks on one of them, pin the previous minor (bits-ui 2.20.x, kit 3.1.x) and retry before debugging further.

### 13.3 Tokens → Tailwind and shadcn

shadcn components use their own variable names, and three of them collide with ours (`--accent`, `--muted`, `--border`). We keep **our** names as the source of truth and point shadcn's *utility names* at them in `@theme inline`. The shadcn components (which we own) need no rewriting.

`src/routes/layout.css` after `init` (replace the generated `:root`, `.dark` and `@theme inline` blocks with this):

```css
@import 'tailwindcss';
@import 'tw-animate-css';
@import '@fontsource-variable/inter';

@custom-variant dark (&:where(.dark, .dark *));

:root {
  /* palette: section 3, light values, unchanged */
  --canvas: #eef0f3; --surface: #ffffff; --sunken: #f5f6f8;
  --line: #dde1e7; --line-strong: #c6ccd5;
  --ink: #14161a; --muted: #5b6270; --faint: #8a919c;
  --accent: #2f5bea; --accent-hover: #2549c4; --accent-soft: #e6ecfd; --on-accent: #ffffff;
  --record: #d93a2b; --record-soft: #fbe7e5;
  --success: #1f8a4c; --success-soft: #e3f3ea;
  --warning: #a86a12; --warning-soft: #fbf0dc;
  --danger: #c4322b; --danger-soft: #fbe6e5;
  --brand: #2f5bea;
  --radius: 6px;
}

.dark {
  /* palette: section 3, dark values, unchanged */
  --canvas: #0b0b0d; --surface: #131316; --sunken: #0f1012;
  --line: #222327; --line-strong: #32343a;
  --ink: #f1f1f3; --muted: #a8adb7; --faint: #7e838f;
  --accent: #7f95ff; --accent-hover: #98a9ff; --accent-soft: #1f2540; --on-accent: #0d1020;
  --record: #ff6f64; --record-soft: #3a1d1b;
  --success: #4fce87; --success-soft: #143022;
  --warning: #e7b45a; --warning-soft: #32260f;
  --danger: #ff6f64; --danger-soft: #3a1d1b;
  --brand: #7f95ff;
}

@theme inline {
  --font-sans: 'Inter Variable', ui-sans-serif, system-ui, sans-serif;
  --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;

  /* our names, as utilities: bg-canvas, text-ink, bg-accent-soft, text-success … */
  --color-canvas: var(--canvas);
  --color-surface: var(--surface);
  --color-sunken: var(--sunken);
  --color-line: var(--line);
  --color-line-strong: var(--line-strong);
  --color-ink: var(--ink);
  --color-faint: var(--faint);
  --color-accent-hover: var(--accent-hover);
  --color-accent-soft: var(--accent-soft);
  --color-on-accent: var(--on-accent);
  --color-record: var(--record);
  --color-record-soft: var(--record-soft);
  --color-success: var(--success);
  --color-success-soft: var(--success-soft);
  --color-warning: var(--warning);
  --color-warning-soft: var(--warning-soft);
  --color-danger: var(--danger);
  --color-danger-soft: var(--danger-soft);
  --color-brand: var(--brand);

  /* shadcn's names, pointed at ours */
  --color-background: var(--surface);
  --color-foreground: var(--ink);
  --color-card: var(--surface);
  --color-card-foreground: var(--ink);
  --color-popover: var(--surface);
  --color-popover-foreground: var(--ink);
  --color-primary: var(--accent);
  --color-primary-foreground: var(--on-accent);
  --color-secondary: var(--sunken);
  --color-secondary-foreground: var(--ink);
  --color-muted: var(--sunken);              /* shadcn: muted = a background */
  --color-muted-foreground: var(--muted);    /* our muted = text */
  --color-accent: var(--sunken);             /* shadcn: accent = hover fill */
  --color-accent-foreground: var(--ink);
  --color-destructive: var(--danger);
  --color-border: var(--line);
  --color-input: var(--faint);            /* control borders, 3:1 (§4) */
  --color-ring: var(--accent);
  --color-sidebar: var(--canvas);
  --color-sidebar-foreground: var(--ink);
  --color-sidebar-primary: var(--accent);
  --color-sidebar-primary-foreground: var(--on-accent);
  --color-sidebar-accent: var(--accent-soft);
  --color-sidebar-accent-foreground: var(--accent);
  --color-sidebar-border: var(--line);
  --color-sidebar-ring: var(--accent);

  --radius-sm: 4px;
  --radius-md: 6px;
  --radius-lg: 6px;
  --radius-xl: 6px;
  --radius-2xl: 6px;
}

@layer base {
  * { @apply border-line outline-ring/50; }
  html { @apply bg-canvas text-ink font-sans; }
  ::placeholder { color: var(--muted); }
  :focus-visible { outline: 2px solid var(--accent); outline-offset: -2px; } /* inside, §11.5 */
}
```

**Rules for code that uses these utilities:**

- **Brand blue.** For the accent blue as text, a border or a fill, write `text-primary`, `border-primary` or `bg-primary`. Never write `bg-accent`, which means the hover fill.
- **Secondary text** is `text-muted-foreground`. `text-faint` is only for icons and line art.
- **State** is `text-success`, `bg-success-soft` and the like, following the section 4 rules.

### 13.4 How code is written

**Today (Svelte 3 and SvelteKit 1).** Keep these conventions in files that haven't migrated yet:

- **Formatting.** Prettier: tabs, single quotes, no trailing commas, print width 100. ESLint: `eslint:recommended` + `@typescript-eslint` + prettier.
- **Files.** File names are kebab-case: `deployment-card.svelte`, `cancel-button.svelte`. Page-local pieces sit next to their route (`app-form.svelte`, `service-url.svelte`); shared ones go in `src/components/`.
- **Aliases.** `$components` → `src/components`, `$assets` → `src/assets`, `$lib` → `src/lib`. Pages never use relative paths into `lib`.
- **Component scripts.** They are `<script lang="ts">` with `export let` props.
  - A `class` prop is re-exported as `export { className as class }`.
  - `$$restProps` passes attributes through, and `type $$Props` types it.
  - `$$Generic` types generic components (`data-table`, `dropdown`).
- **Svelte syntax.** Reactivity is `$:`. Events are `on:click` and are forwarded with a bare `on:click`. Content uses slots, including named slots and `let:` props (`<Form let:submitting let:errors>`).
- **Styles.** Each component has a `<style module>` (CSS modules) using the `--co-*`, `--sp-*`, `--ty-*` and `--ra-*` tokens. `bind(prop)` puts a prop into CSS (`stack.svelte`).
- **User-facing strings.** Every string is a typed translation key. Props take an `AppTranslationsString` (`text="app.new"`), and the component calls `l.translate`. Add each key to both `en.ts` and `fr.ts`.
- **Data.** `+page.ts` and `+layout.ts` pre-fetch with `service.fetchX(…, { fetch, depends })`. The `.svelte` page then subscribes with `service.queryX()` for live, polled data. Mutations go through a service, which invalidates the right URLs.
- **Forms.** `<Form handler={submit}>`, with the `submit` function from `submitter()`. Field-level API errors go in `remoteError={errors?.field}`.
- **Types.** Global helpers are declared in `app.d.ts`: `Maybe<T>`, `Patch<T>` (`undefined` = unchanged, `null` = remove), `KeysOfType`, `DateValue`, `HtmlInputType`.
- **Comments** explain *why*, and `FIXME:` marks known debt.
- **Tests** are `*.test.ts` next to the code, written in vitest with `describe` and `it`.

**After migration (Svelte 5 and SvelteKit 3).** New and migrated code follows these rules:

- **Props.** Use `let { data, class: className, ...rest }: Props = $props();`, with a `type Props` above. Two-way props use `$bindable()`.
- **Reactivity.** State is `$state`, computed values are `$derived` and side effects are `$effect`. Never write a `$:` label.
- **Events.** Events are attributes: `onclick={…}`, and callback props such as `onOpenChange`. Never `on:`.
- **Content.** Use snippets: `{#snippet name(args)}` and `{@render children?.()}`. Never slots or `let:`.
- **Styling.** Use Tailwind classes merged with `cn()` from `$lib/utils` (or `#lib/utils.js` once the Kit 3 alias task runs). Component variants use `tailwind-variants`, as shadcn does. No `<style>` blocks unless a style can't be a utility.
- **File layout.**
  - Generated UI goes in `src/lib/components/ui/<name>/` with an `index.ts` and is imported as `import * as Card from '$lib/components/ui/card/index.js'`.
  - Domain components (`app-card`, `deployment-card` …) go in `src/lib/components/` and import from `ui/`.
  - Page-local components stay next to their route.
- **Data layer.** Keep `fetcher`, `resources`, `submitter` and `localization` as they are. Stores still work in Svelte 5, so `$data` keeps working. Move a store to a `.svelte.ts` class with `$state` only when you touch that file for another reason.
- **Kit modules.** Use `$app/state` (`page`, `navigating`) instead of `$app/stores`; the Kit 3 `app-state` task does this. Use `$app/env` instead of `$app/environment`.
- **Accessibility.** Use Bits UI parts for anything that opens, closes, selects or traps focus. Never hand-roll a menu with a checkbox.

### 13.5 Migration order

**Status (2026-11-14): steps 1 to 7 are done.** The framework is upgraded, Tailwind, shadcn-svelte and the components are installed, and the theme and providers are in place.

- **What changed from the plan:**
  - The legacy `sv migrate` steps (`svelte-4`, `sveltekit-2`, `svelte-5`) need an interactive terminal. The version bumps were done by hand instead, with the few code fixes Svelte 5 and SvelteKit 2 needed.
  - Components still use the Svelte 4 syntax, which Svelte 5 runs in legacy mode. Only `src/routes/+layout.svelte` and `(main)/+layout.svelte` use runes.
  - The old global CSS (`the-new-css-reset` and `src/assets/styles.css`) was kept in a `legacy` cascade layer instead of being removed (see `src/routes/layout.css`).
  - `<ModeWatcher defaultMode="dark">` keeps the dark-only old pages looking as before.
- **Step 9 has started:** the sign-in page is migrated (2026-11-14), with the auth layout, the `DitherImage` component (`src/lib/components/dither-image.svelte`, maths in `src/lib/dither.ts`) and the button loading state. The generated components were edited for DESIGN.md: inset focus rings, no shadows, sunken inputs with `faint` borders, the soft destructive `Alert`.
- **Step 8 is done (2026-11-14):** the shell (§5, Shell). The old `topbar`, `account`, `bottom-bar` and `progress-bar` are deleted.
- **Rebuilt pages:** every page. Sign in, Applications, Application, New application and Settings, New deployment, Deployment, Targets, New target and Target settings, Registries (with New and edit), Jobs, Profile and the error pages. Destructive and changing actions confirm in `ConfirmDialog` (`#lib/components/confirm-dialog.svelte`), which replaces `window.confirm`; deleting an application, a target or a registry asks for its name. Drawers, drop zones and highlighted code follow §11.6.
- **End-to-end tests** (`npm run test:e2e`, `make test-e2e`): Playwright against the real Go server and a fresh database; `e2e/auth.test.ts`, `e2e/shell.test.ts` and `e2e/pages.test.ts`.
- **Next:** step 10. No route imports `src/components/` any more; delete it with the legacy CSS layer, then check the pages still look the same.

Make each step its own commit, and make sure `npm run check`, `npm test`, `npm run build` and `make build` pass before starting the next. After any frontend build, restore `cmd/serve/front/build/.keep`, because the build deletes it:

```sh
git checkout HEAD -- cmd/serve/front/build/.keep
```

Run every command below in `cmd/serve/front` unless the step names another folder.

1. **Node.** Bump the Dockerfile to `FROM node:24-alpine` and CI to `node-version: 24`.
2. **Svelte 3 → 4 and SvelteKit 1 → 2** (legacy migrations).

   ```sh
   npx sv@1.3.0 migrate svelte-4
   npx sv@1.3.0 migrate sveltekit-2
   ```

   Fix `throw error()`/`redirect()` (no `throw` in Kit 2), the `goto` external URLs and the `cookies` path. Upgrade to the last 2.x (`@sveltejs/kit@2.70.4`, `adapter-static@3.0.11`) to get the deprecation warnings.
3. **Svelte 4 → 5.**

   ```sh
   npx sv@1.3.0 migrate svelte-5
   ```

   It rewrites `export let` → `$props`, `$:` → `$derived`/`$effect`, `on:` → attributes and slots → snippets where it can. Search for `@migration-task` comments and fix them by hand.
   - `svelte-preprocess-cssmodules` 3.0.1 supports Svelte 5, so `<style module>` keeps working during the transition.
   - `$$Generic` becomes `<script lang="ts" generics="T extends …">`.
4. **SvelteKit 2 → 3.** Run each task alone and commit it alone:

   ```sh
   npx sv@1.3.0 migrate sveltekit-3 --tasks package-json   # also tsconfig (prerequisites)
   npx sv@1.3.0 migrate sveltekit-3 --tasks svelte-config  # svelte.config.js → vite.config.ts
   npx sv@1.3.0 migrate sveltekit-3 --tasks environment
   npx sv@1.3.0 migrate sveltekit-3 --tasks app-state
   ```

   - The `kit.alias` entries (`$components`, `$assets`) move into the `sveltekit({ … })` plugin options, or into `package.json` `imports` (`#components/*`).
   - TypeScript lands on 6.0.x.
5. **Tooling.** Move to ESLint 10 flat config (`eslint.config.js` with `eslint-plugin-svelte` + `typescript-eslint`) and Prettier 3 with `prettier-plugin-svelte` 4 and `prettier-plugin-tailwindcss`. Run vitest 5 and svelte-check 4.
6. **Tailwind + shadcn-svelte + the UI dependencies.** This is the one CLI step, below; it needs steps 2 to 4 first.
7. **Theme.** Replace the generated CSS with section 13.3. Add `<ModeWatcher />` and `<Toaster />` in `src/routes/+layout.svelte`, and drop `the-new-css-reset` and the `--co-*` file once nothing uses them.
8. **Shell.** Build the `Sidebar`, the header and the mobile `Sheet`, and retire `topbar`, `account` and `bottom-bar`.
9. **Pages, one per commit.** Start with the simplest and end with the deployment page: Sign in → Registries → Targets → Applications → App form → New deployment → Deployment → Jobs → Profile → errors. Swap components per the 11.2 table. Delete an old component once nothing imports it.
10. **Clean-up.** Remove `svelte-preprocess`, `svelte-preprocess-cssmodules`, `stack.svelte` and `src/assets/styles.css`. Update AGENT.md §6 (frontend) to match.

### 13.6 The one CLI install (step 6)

Run it from `cmd/serve/front` **after** the framework migration (steps 2 to 4). Bits UI and shadcn-svelte refuse Svelte 3.

This is the command that was run (2026-11-14). It installs everything in §13.2 and every component in §12:

```sh
npx sv@1.3.0 add "tailwindcss=plugins:none" --no-git-check --install npm \
&& npx svelte-kit sync \
&& npx shadcn-svelte@1.8.1 init --preset vega --base-color neutral --css src/routes/layout.css \
     --lib-alias '#lib' --components-alias '#lib/components' --utils-alias '#lib/utils' \
     --hooks-alias '#lib/hooks' --ui-alias '#lib/components/ui' \
&& npx svelte-kit sync \
&& npx shadcn-svelte@1.8.1 add -y \
     button input textarea label field input-group select checkbox switch \
     card badge alert alert-dialog dialog sheet dropdown-menu tooltip \
     sidebar breadcrumb table pagination collapsible scroll-area separator \
     skeleton spinner empty item avatar sonner kbd \
&& npm rm @fontsource/inter
```

Notes on this command:

- **What `init` installs.** `init` with the **Vega** preset (Lucide + Inter, the classic shadcn look) brings in bits-ui, @internationalized/date, @lucide/svelte, mode-watcher, svelte-sonner, @fontsource-variable/inter, tailwind-variants, tw-animate-css, `cn` and `shadcn-svelte` itself. `shadcn-svelte` is installed because `layout.css` imports `shadcn-svelte/tailwind.css`. All of these were then pinned to exact versions.
- **The preset is required.** Without `--preset`, `init` asks for a preset in an interactive prompt. The presets are Nova, Vega, Maia, Lyra, Mira, Luma, Sera and Rhea.
- **`npx svelte-kit sync` between the steps is required on npm with SvelteKit 3.** npm removes the generated `node_modules/$app` on every install, and `shadcn-svelte add` then fails with "File '$app/tsconfig' not found" (upstream issue #2953).
- **The generated CSS was replaced.** After `init`, the generated `:root`, `.dark` and `@theme inline` blocks were replaced with §13.3, adding the `legacy` layer for the old global CSS.
- **Re-adding components.** To re-add a component after a shadcn-svelte update, run `npx shadcn-svelte@latest add <name> -o`. Then re-apply our edits, such as the 4px badge corner.

---

## 14. Sources

- [shadcn-svelte docs index](https://www.shadcn-svelte.com/llms.txt), [SvelteKit installation](https://www.shadcn-svelte.com/docs/installation/sveltekit), [CLI](https://shadcn-svelte.com/docs/cli.md), [theming](https://shadcn-svelte.com/docs/theming.md), [components.json](https://shadcn-svelte.com/docs/components-json.md), [changelog](https://shadcn-svelte.com/docs/changelog.md)
- shadcn-svelte SvelteKit 3 issues: [#2849](https://github.com/huntabyte/shadcn-svelte/issues/2849), [#2953](https://github.com/huntabyte/shadcn-svelte/issues/2953), [#2915](https://github.com/huntabyte/shadcn-svelte/issues/2915), [#2917](https://github.com/huntabyte/shadcn-svelte/issues/2917)
- [`sv migrate`](https://svelte.dev/docs/cli/sv-migrate), [Migrating to SvelteKit 3](https://svelte.dev/docs/kit/migrating-to-sveltekit-3), [Migrating to SvelteKit 2](https://svelte.dev/docs/kit/migrating-to-sveltekit-2), [`sv add tailwindcss`](https://svelte.dev/docs/cli/tailwind)
- [Bits UI on npm](https://www.npmjs.com/package/bits-ui), [Bits UI vs Melt UI (2026)](https://sveltekitui.com/blog/bits-ui-vs-melt-ui/), [Svelte 4 → 5 UI library compatibility](https://sveltekitui.com/blog/migrating-svelte-4-to-5-ui-libraries/)
- Version numbers: `npm view <package> version` and `peerDependencies`, 2026-11-14
