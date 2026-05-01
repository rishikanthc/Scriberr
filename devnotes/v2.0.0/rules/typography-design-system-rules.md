# Scriberr Typography Design System (DM Sans)

## Purpose

This document defines a centralized typography system for Scriberr so hierarchy is explicit, scanning is faster, and typography remains cohesive across transcript, notes, and product surfaces.

## Foundation

- Primary font family: `DM Sans` from Google Fonts.
- Loading strategy: static local `@font-face` files in `web/frontend/public/fonts/dm-sans`.
- Imported static weights: `400`, `500`, `600`, `700`, with matching italics.
- Do not use range-based variable font imports for the app shell; browser font inspectors may expose the variable font's named instance instead of the active CSS weight.
- The app root uses shared weight tokens for `400`, `500`, `600`, and `700`.
- Single UI family across headings, body, metadata, controls, and note surfaces.
- Serif reading utility remains optional for long-form reading experiments only.

## Hierarchy Principles

- Use scale and spacing before decoration.
- Keep heading contrast obvious: larger size + stronger weight + tighter line-height.
- Keep body text highly legible: medium weight + comfortable line-height.
- Keep metadata quieter: smaller size + secondary/tertiary color.
- Avoid one-off font sizes inside components unless product-critical.

## Central Token Model

All typography should consume shared CSS variables from `design-system.css`.

Required token groups:

- Family:
  - `--scr-font-sans`
- Weights:
  - `--scr-font-weight-regular`
  - `--scr-font-weight-medium`
  - `--scr-font-weight-semibold`
  - `--scr-font-weight-bold`
- Sizes:
  - `--scr-type-display-xl`
  - `--scr-type-heading-lg`
  - `--scr-type-heading-md`
  - `--scr-type-body-lg`
  - `--scr-type-body-md`
  - `--scr-type-body-sm`
  - `--scr-type-meta`
  - `--scr-type-micro`
- Leading:
  - `--scr-line-tight`
  - `--scr-line-base`
  - `--scr-line-relaxed`

## Recommended Mapping

- Page/recording title: `display-xl` + `semibold` + `text-primary`.
- Section heading: `heading-md` + `semibold` + `text-primary`.
- Body/default: `body-md` + `medium` + `text-primary`.
- Supporting copy: `body-sm` + `medium` + `text-secondary`.
- Metadata/chips/timestamps: `meta` + `medium` + `text-secondary`.
- Tiny labels/help/status: `micro` + `medium` + `text-tertiary`.

## Notes Sidebar Typography Rules

- Quote line should read as context: `heading-lg` on desktop, never bold-heavy.
- Note bubble text should be compact and readable: `body-lg` with relaxed leading.
- Reply placeholder/input should be one step quieter than note content.
- Timestamp and count chips must stay compact and secondary.

## Consistency Rules

- New components must consume tokens; do not hardcode random font-size values.
- Reuse existing semantic classes before adding new typography variants.
- If a size is reused 3+ times, promote it to a token.
- Typography adjustments should be made in token layer first, then component overrides.

## Accessibility and Readability

- Minimum body size for dense product surfaces: `0.875rem`.
- Maintain clear contrast between primary and secondary text colors.
- Avoid extreme font-weight jumps between adjacent levels.
- Keep line lengths and vertical rhythm stable to prevent visual jitter.
