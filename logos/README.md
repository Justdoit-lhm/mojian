# Mojian logo

**Ink & Cursor.** The mark is the first letter of the name, set in Literata and
followed by a vermilion block cursor — the one symbol every editor shares.
Everything else is ink on paper; the cursor is the only loud element in the
system. The wordmark is lowercase to keep the tone quiet, and it is outlined so
no font is needed to use it.

Open [`preview.html`](preview.html) for the full presentation; a flat snapshot
is in [`png/preview.png`](png/preview.png).

## Files

### SVG masters

| File | Use |
|---|---|
| `svg/mojian-logo-horizontal.svg` | Primary lockup: mark + wordmark, side by side |
| `svg/mojian-logo-vertical.svg` | Stacked lockup: mark above wordmark |
| `svg/mojian-mark.svg` | Mark only — transparent background |
| `svg/mojian-wordmark.svg` | Wordmark only — transparent background |
| `svg/mojian-app-icon.svg` | App icon, 1024 master with paper tile |
| `svg/mojian-app-icon-mono.svg` | App icon, ink tile / paper mark |
| `svg/mojian-app-icon-reversed.svg` | App icon on dark, lighter cursor |
| `svg/mojian-logo-horizontal-mono.svg` | Horizontal lockup, one ink color |
| `svg/mojian-logo-horizontal-reversed.svg` | Horizontal lockup for dark backgrounds |
| `svg/mojian-logo-vertical-mono.svg` | Stacked lockup, one ink color |
| `svg/mojian-logo-vertical-reversed.svg` | Stacked lockup for dark backgrounds |
| `svg/mojian-mark-mono.svg` | Mark, one ink color |
| `svg/mojian-mark-reversed.svg` | Mark for dark backgrounds |
| `svg/mojian-wordmark-reversed.svg` | Wordmark for dark backgrounds |

### PNG exports

| File | Size |
|---|---|
| `png/mojian-app-icon-1024.png` | 1024 × 1024, app icon master |
| `png/mojian-app-icon-512.png` … `-16.png` | 512, 256, 128, 64, 32, 16 |
| `png/mojian-logo-horizontal-1600.png` | 1600 × 382 |
| `png/mojian-logo-horizontal-reversed-1600.png` | 1600 × 382 |
| `png/mojian-logo-vertical-900.png` | 900 × 546 |

## Palette

| Name | Hex | Role |
|---|---|---|
| Ink | `#1F1B16` | Mark, wordmark, monochrome |
| Paper | `#FBF7EF` | Tile base, reversed mark |
| Vermilion | `#D24A2B` | The cursor — the only accent |
| Blush | `#FAEFE9` | Icon gradient, mid stop |
| Deep paper | `#F1E2CB` | Icon gradient, end stop |
| Reversed cursor | `#E4653F` | Cursor on ink backgrounds |

The icon gradient runs `#FFFCF8 → #FAEFE9 → #F1E2CB` at 135°.

## Typography

Literata (SIL Open Font License 1.1), outlined in the SVG files:

- Mark: Literata 600.
- Wordmark: Literata 550, tracking −16/1000 em.

The wordmark is supplied as outlines, not live text. Do not reset it in another
typeface; use `svg/mojian-wordmark.svg` or the full lockups.

## Clear space

Keep at least the **height of the mark's `m`** (its x-height) clear on every
side. In the horizontal lockup that is roughly the height of the lowercase `m`;
in the app icon the paper tile already provides the margin. No type, rules, or
image edges inside that space.

## Minimum sizes

| Asset | Minimum |
|---|---|
| Horizontal lockup | 100 px wide (28 mm print) |
| Vertical lockup | 120 px wide |
| Wordmark | 80 px wide |
| Mark | 24 px high |
| App icon | 16 px; 32 px or larger recommended |

## Don'ts

- Do not recolor the cursor. It is the only vermilion element.
- Do not separate the `m` from the cursor, or reorder them.
- Do not stretch, rotate, skew, or add shadows, glows, or outlines.
- Do not place the color mark on a busy photo without the paper tile and clear
  space.
- Do not set the wordmark in another typeface or re-space it.
- Do not replace the app icon's superellipse with a plain rounded rectangle.

## App icon

The app's icon source, `resources/icon.png`, now uses
`logos/png/mojian-app-icon-1024.png`. A rebuild embeds it:

```sh
mygo build
```

To restore or copy it elsewhere:

```sh
cp logos/png/mojian-app-icon-1024.png resources/icon.png
```

The source icon keeps roughly a 4% transparent margin to match the previous
asset. Its shape is a superellipse (n = 5), not a circular rounded
rectangle. The SVG is the master; the PNGs are exports.

## Exporting

The PNGs in `png/` were rendered from the SVGs with Chromium and downsampled
with Lanczos. To re-export an SVG at another size:

```sh
# macOS Quick Look (simple, no install)
qlmanage -t -s 1024 -o out/ logos/svg/mojian-app-icon.svg

# ImageMagick / librsvg
rsvg-convert -w 1024 -h 1024 logos/svg/mojian-app-icon.svg > out/icon.png
```

Keep the transparent margin when exporting the app icon.

## License

The logo files are part of the Mojian project. The wordmark letterforms are
outlined from Literata, copyright 2017 The Literata Project Authors, designed
by Veronika Burian and José Scaglione, licensed under the SIL Open Font License
1.1. No font files are redistributed here.
