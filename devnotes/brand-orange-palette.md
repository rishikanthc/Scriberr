# Scriberr Brand Orange Palette

## Source Colors

These orange colors are present in the current logo and frontend styles.

| Token | Hex | Source |
| --- | --- | --- |
| `--scr-color-orange-500` | `#FF9800` | `logo.svg` gradient start |
| `--scr-color-orange-700` | `#FF3D00` | `logo.svg` gradient end |
| `--scr-brand-solid` | `#FF6D20` | Existing `--brand-solid` midpoint |
| `--brand-50` | `#fff8e6` | Existing `src/index.css` brand ramp |
| `--brand-100` | `#ffedcc` | Existing `src/index.css` brand ramp |
| `--brand-200` | `#ffdd99` | Existing `src/index.css` brand ramp |
| `--brand-300` | `#ffcc66` | Existing `src/index.css` brand ramp |
| `--brand-400` | `#ffbb33` | Existing `src/index.css` brand ramp |
| `--brand-500` | `#fe9a00` | Existing `src/index.css` brand ramp |
| `--brand-600` | `#e68a00` | Existing `src/index.css` brand ramp |
| `--brand-700` | `#cc7a00` | Existing `src/index.css` brand ramp |
| `--brand-800` | `#b36b00` | Existing `src/index.css` brand ramp |
| `--brand-900` | `#995c00` | Existing `src/index.css` brand ramp |
| `--brand-950` | `#663d00` | Existing `src/index.css` brand ramp |
| legacy gradient start | `#FFAB40` | Existing commented/older gradient and button usage |

## New Frontend Token Direction

The new frontend foundation uses `web/frontend/src/styles/design-system.css`.

Core brand tokens:

```css
--scr-brand-gradient: linear-gradient(135deg, #FF9800 0%, #FF3D00 100%);
--scr-brand-solid: #FF6D20;
--scr-brand-ink: #FF3D00;
--scr-brand-muted: color-mix(in srgb, var(--scr-brand-solid) 11%, transparent);
--scr-brand-border: color-mix(in srgb, var(--scr-brand-solid) 24%, transparent);
```

Guidelines:

- Use orange for primary actions, active navigation, and recording/file identity.
- Keep light mode white-based and dark mode black-based.
- Do not hard-code orange hex values in React components. Reference semantic tokens or atomic UI classes.
- Keep status colors separate from brand orange so completion, failure, and queued states stay readable.
