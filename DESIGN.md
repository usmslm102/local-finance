---
name: LocalFinance
description: A restrained private workspace for recorded finances and explicit plans.
colors:
  background: "oklch(1 0 0)"
  foreground: "oklch(0.145 0 0)"
  card: "oklch(1 0 0)"
  card-foreground: "oklch(0.145 0 0)"
  popover: "oklch(1 0 0)"
  popover-foreground: "oklch(0.145 0 0)"
  primary: "oklch(0.42 0.11 160)"
  primary-foreground: "oklch(0.985 0 0)"
  secondary: "oklch(0.97 0 0)"
  secondary-foreground: "oklch(0.205 0 0)"
  muted: "oklch(0.97 0 0)"
  muted-foreground: "oklch(0.48 0 0)"
  success: "oklch(0.42 0.11 160)"
  warning: "oklch(0.43 0.1 65)"
  info: "oklch(0.43 0.12 255)"
  accent: "oklch(0.97 0 0)"
  accent-foreground: "oklch(0.205 0 0)"
  destructive: "oklch(0.577 0.245 27.325)"
  destructive-foreground: "oklch(0.98 0 0)"
  border: "oklch(0.922 0 0)"
  input: "oklch(0.922 0 0)"
  ring: "oklch(0.708 0 0)"
  chart-1: "oklch(0.87 0 0)"
  chart-2: "oklch(0.556 0 0)"
  chart-3: "oklch(0.439 0 0)"
  chart-4: "oklch(0.371 0 0)"
  chart-5: "oklch(0.269 0 0)"
  sidebar: "oklch(0.985 0 0)"
  sidebar-foreground: "oklch(0.145 0 0)"
  sidebar-primary-foreground: "oklch(0.985 0 0)"
  sidebar-accent: "oklch(0.97 0 0)"
  sidebar-accent-foreground: "oklch(0.205 0 0)"
  sidebar-border: "oklch(0.922 0 0)"
  sidebar-ring: "oklch(0.708 0 0)"
  dark-background: "oklch(0.145 0 0)"
  dark-foreground: "oklch(0.985 0 0)"
  dark-card: "oklch(0.205 0 0)"
  dark-card-foreground: "oklch(0.985 0 0)"
  dark-popover: "oklch(0.205 0 0)"
  dark-popover-foreground: "oklch(0.985 0 0)"
  dark-primary: "oklch(0.77 0.14 160)"
  dark-primary-foreground: "oklch(0.18 0.03 160)"
  dark-secondary: "oklch(0.269 0 0)"
  dark-secondary-foreground: "oklch(0.985 0 0)"
  dark-muted: "oklch(0.269 0 0)"
  dark-muted-foreground: "oklch(0.708 0 0)"
  dark-success: "oklch(0.77 0.14 160)"
  dark-warning: "oklch(0.8 0.12 80)"
  dark-info: "oklch(0.76 0.1 255)"
  dark-accent: "oklch(0.269 0 0)"
  dark-accent-foreground: "oklch(0.985 0 0)"
  dark-destructive: "oklch(0.704 0.191 22.216)"
  dark-destructive-foreground: "oklch(0.98 0 0)"
  dark-border: "oklch(1 0 0 / 10%)"
  dark-input: "oklch(1 0 0 / 15%)"
  dark-ring: "oklch(0.556 0 0)"
  dark-chart-1: "oklch(0.87 0 0)"
  dark-chart-2: "oklch(0.556 0 0)"
  dark-chart-3: "oklch(0.439 0 0)"
  dark-chart-4: "oklch(0.371 0 0)"
  dark-chart-5: "oklch(0.269 0 0)"
  dark-sidebar: "oklch(0.205 0 0)"
  dark-sidebar-foreground: "oklch(0.985 0 0)"
  dark-sidebar-primary-foreground: "oklch(0.985 0 0)"
  dark-sidebar-accent: "oklch(0.269 0 0)"
  dark-sidebar-accent-foreground: "oklch(0.985 0 0)"
  dark-sidebar-border: "oklch(1 0 0 / 10%)"
  dark-sidebar-ring: "oklch(0.556 0 0)"
typography:
  headline:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 700
    lineHeight: "2rem"
    letterSpacing: "-0.025em"
  section-title:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: "1.75rem"
    letterSpacing: "normal"
  title:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "1rem"
    fontWeight: 500
    lineHeight: "1.375"
    letterSpacing: "normal"
  body:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: "1.25rem"
    letterSpacing: "normal"
  label:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 500
    lineHeight: "1.25rem"
    letterSpacing: "normal"
  caption:
    fontFamily: "Inter Variable, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: "1rem"
    letterSpacing: "normal"
rounded:
  sm: "0.375rem"
  md: "0.5rem"
  lg: "0.625rem"
  xl: "0.875rem"
  4xl: "1.625rem"
spacing:
  1: "0.25rem"
  2: "0.5rem"
  3: "0.75rem"
  4: "1rem"
  5: "1.25rem"
  6: "1.5rem"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    rounded: "{rounded.lg}"
    typography: "{typography.label}"
    height: "2rem"
    padding: "0 0.625rem"
  button-outline:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.lg}"
    height: "2rem"
    padding: "0 0.625rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.lg}"
    height: "2rem"
    padding: "0 0.625rem"
  input:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.lg}"
    height: "2rem"
    padding: "0.25rem 0.625rem"
  card:
    backgroundColor: "{colors.card}"
    textColor: "{colors.card-foreground}"
    rounded: "{rounded.xl}"
    padding: "1rem"
  badge-outline:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.4xl}"
    height: "1.25rem"
    padding: "0.125rem 0.5rem"
  sidebar-item:
    backgroundColor: "transparent"
    textColor: "{colors.muted-foreground}"
    rounded: "{rounded.md}"
    height: "2.25rem"
    padding: "0 0.625rem"
  progress:
    backgroundColor: "{colors.muted}"
    height: "0.25rem"
    width: "100%"
---
# Design System: LocalFinance

## Overview

**Creative North Star: "The Private Ledger"**

LocalFinance uses the familiar order of a private ledger: restrained neutral surfaces, clear financial figures, and an emerald accent for actions and positive state. The interface helps people review local records and turn them into explicit plans, with controls and explanations close to the data they affect.

The inherited visual system is shadcn Base UI, base-nova with the neutral base, bundled Inter Variable, Lucide icons, and semantic CSS variables. Its character is compact, calm, and operational. Dark mode preserves the same roles; privacy presentation and accessible state feedback are part of the interface rather than decoration.

**Key Characteristics:**
- Neutral working surfaces with purposeful emerald accents.
- Compact typography and tabular financial numbers.
- Labeled controls, visible state feedback, and local-data context.
- Responsive stacking and comfortable coarse-pointer controls.

## Colors

A restrained neutral palette gives the emerald accent and semantic status colors clear purpose. The frontmatter preserves the canonical OKLCH values from `frontend/src/index.css`; `dark-` entries describe the corresponding dark-theme overrides.

### Primary

- **Ledger Emerald** (`primary`): primary buttons, selected navigation icons, progress fills, and positive indicators. The brighter dark counterpart keeps the role readable on charcoal surfaces.
- **Primary Ink** (`primary-foreground`): contrast text on filled primary controls.

### Neutral

- **Paper / Charcoal** (`background`, `card`, `popover`, and dark counterparts): the workspace, bounded working regions, and elevated surfaces.
- **Ledger Ink** (`foreground`): titles, values, and primary reading content.
- **Soft Neutral** (`secondary`, `muted`, `accent`): supporting surfaces and hover/selected state. These are neutral roles, not additional brand accents.
- **Quiet Ink** (`muted-foreground`): descriptions and supporting labels.
- **Fine Boundary** (`border`, `input`, `ring`): separators, field strokes, and focus feedback.
- **Sidebar Neutral** (`sidebar` and sidebar role tokens): the app shell and its navigation states.
- **Graphite Series** (`chart-1` through `chart-5`): the shared neutral chart palette. Individual charts can map semantic primary and muted roles to meaningful series.

Status roles supplement the primary/neutral palette: `success` shares the emerald role; `warning` is warm ochre, `info` is blue, and `destructive` is red. They communicate state rather than form a decorative secondary palette.

**The Semantic Accent Rule.** Use the primary accent for primary actions, active navigation icons, and positive financial state. Use warning, info, and destructive roles for their corresponding meaning, with text or icons alongside color.

## Typography

**Body and Heading Font:** bundled Inter Variable, with sans-serif fallback. Headings and interface copy share a family. Monospace is reserved for keyboard hints and masked-value treatment.

**Character:** concise and legible. Weight and spacing establish hierarchy without a separate display typeface.

### Hierarchy

- **Headline:** page titles use the frontmatter headline role with tight tracking.
- **Section Title:** planning section headings use the section-title role.
- **Title:** card headings use the medium title role and snug leading.
- **Body:** descriptions and operating copy use the body role; longer explanations use a prose-width container where present.
- **Label:** field and button labels use the medium label role.
- **Caption:** compact supporting information uses the caption role. Sidebar group labels use a smaller uppercase, tracked treatment.

Inputs use a larger text size on small screens, returning to compact body sizing at the medium breakpoint. Financial totals can use the headline size and semibold weight.

**The Financial Precision Rule.** Use tabular numerals for financial comparisons. Compact chart labels can reduce crowding; tables and tooltips retain precise INR values when privacy mode permits.

## Layout

The shared sidebar and header organize the app, while page titles, descriptions, and controls establish the local task. Page sections commonly use a vertical gap of the spacing-6 token, with spacing-4 and spacing-3 for tighter internal grouping. Cards use spacing-4 internal padding by default and spacing-3 in their small variant.

Layouts respond to available space rather than fixing financial content to a desktop canvas. In the established planning view, assumptions and the scenario chart form a one-to-two grid at the large breakpoint; they stack below it. Goal cards use two columns at the medium breakpoint and one below. Wrapping header actions and scrollable tables preserve access to dense content. These are observed surface examples, not required compositions for every page.

Dialogs fit within the viewport with a one-rem outer margin, a maximum height based on the dynamic viewport, and internal vertical scrolling. Their actions stack on narrow screens and align horizontally from the small breakpoint. Coarse-pointer controls expand to at least 2.75rem high, with the same minimum width for labeled icon buttons.

## Elevation & Depth

Depth is predominantly tonal: cards have a fine foreground-tinted ring instead of a heavy drop shadow. Active sidebar entries use a small shadow. Overlays add a light black scrim and subtle backdrop blur; dialogs retain the same fine ring and neutral surface as the underlying interface.

**The Tonal Surface Rule.** Separate working regions with neutral tone, a fine border or ring, and spacing. Reserve stronger depth and backdrop treatment for overlays.

## Shapes

The base radius is the frontmatter lg token. Fields and default buttons share that gentle curve; cards and dialogs use xl. Compact sidebar items use md, while badges use the more rounded 4xl shape. Progress tracks are fully rounded. Borders stay thin, with explicit focus and invalid state treatments rather than decorative edge effects.

## Components

### Buttons

Compact, clear actions. Default buttons are filled emerald; outline and ghost controls support secondary tasks. Default height and horizontal padding are recorded in frontmatter. Hover changes the fill or neutral tone. Focus adds a ring-colored border and a three-pixel translucent ring. Disabled controls fade and reject pointer interaction; destructive controls use a tinted red surface. Ordinary pressed buttons move down slightly, with spatial motion curtailed under reduced-motion preferences.

### Cards / Containers

Quiet working regions with xl corners, card foreground/background roles, and a fine one-pixel ring. Header and content share the same padding rhythm; footers can use a muted tint and top border. Use these existing components for bounded content groups rather than inventing a new surface vocabulary.

### Inputs / Fields

Labeled native fields with gently curved input strokes, compact desktop sizing, and larger small-screen text. Focus uses the same three-pixel ring language as buttons. Invalid fields receive destructive border/ring treatment; disabled fields fade and use a tinted input background. Descriptions explain assumptions and consequences, while error and status text remains tied to the affected operation.

### Navigation

Sidebar items pair Lucide icons with compact medium-weight labels. Active entries gain a neutral tint, semibold text, a small shadow, and a primary-colored icon. Hover uses a softer sidebar tint. The shell supports collapsed and mobile navigation; maintain meaningful labels when visible text is reduced. Tabs group views within a page without introducing another navigation identity.

### Badges / Progress

Badges are compact rounded labels, with filled, outline, secondary, and destructive variants. Progress uses a thin muted track and primary fill; a value or completion label gives the visual bar meaning. Do not rely on the fill alone to explain progress.

### Financial Charts and Tables

Charts use the shared container, tooltip, and legend primitives. Planning comparisons use semantic series, compact INR axes, a zero reference where meaningful, and precise right-aligned tabular figures in the companion table. Financial privacy mode replaces revealing chart content with a neutral explanatory region and masks sensitive values; editable financial fields are disabled in the corresponding discreet view.

Import history places “View imported records” and the available “Preview PDF” action together beneath each filename, using outline controls that wrap on narrow screens. The URL-scoped transaction ledger keeps the source filename and an explanation of current ledger edits in a neutral alert, with wrapping actions for PDF preview, import history, and removing the import filter. The PDF dialog also links to imported records; navigation links styled as buttons retain `data-slot="button"` so the shared 44px coarse-pointer target rule applies.

### Dialogs and Feedback

Dialog title, description, close action, and viewport-fitting scroll behavior are supplied by the Base UI-backed primitives. Feedback uses existing alert, empty, loading, and retry patterns. Keep the action required to proceed close to the explanation. Overlay state changes are brief; reduced-motion preferences remove spatial animation while preserving state feedback.

## Do's and Don'ts

### Do:

- **Do** use existing semantic tokens and shadcn Base UI primitives.
- **Do** keep labels, descriptions, errors, and loading feedback attached to the relevant control or region.
- **Do** preserve light/dark role mappings, keyboard focus, reduced-motion behavior, and coarse-pointer target expansion.
- **Do** route financial labels, tables, charts, and editable values through the established privacy treatment.
- **Do** keep chart summaries readable and provide precise tabular comparisons.

### Don't:

- **Don't** introduce remotely loaded fonts or imagery into the offline interface.
- **Don't** use color as the only indication of health, completion, or error.
- **Don't** expose sensitive financial figures through chart labels or tooltips while the corresponding privacy view hides them.
- **Don't** replace established semantic roles with hard-coded light-only colors.
