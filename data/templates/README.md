# OMR Templates

A template is a single JSON file that describes everything the scanner needs to process a specific bubble sheet: the output dimensions, anchor points for perspective correction, image processing configuration, and the location of every answer bubble on the form.

## Usage

```
omr scan <image> <template.json>
omr scan <image> <template.json> --output result.jpg
omr scan <image> <template.json> --display
```

## Example Directory Structure

```
my_template/
├── template.json
└── anchors/
    ├── logo.jpg
    ├── footer.jpg
    └── info.jpg
```

Anchor image paths in the JSON are resolved relative to the JSON file's location.

## Field Reference

### Top-level

| Field       | Type   | Description |
|-------------|--------|-------------|
| `width`     | int    | Output image width in pixels after warping |
| `height`    | int    | Output image height in pixels after warping |
| `anchors`   | array  | Exactly 3 anchor definitions (see below) |
| `config`    | object | Processing parameters (see below) |
| `questions` | array  | Ordered list of questions and their bubble locations (see below) |

### Anchor

Each anchor is a distinctive region of the sheet used to compute the perspective transform. All coordinates are in output (`width` x `height`) space. Choose anchors in distinct corners of the page for the most accurate warp.

| Field    | Type   | Description |
|----------|--------|-------------|
| `path`   | string | Path to a cropped image of the anchor region, relative to the JSON file |
| `center` | Point  | Where the center of this anchor should land in the output (`{"X": n, "Y": n}`) |
| `roi`    | Rect   | Search region in output space (`{"Min": {"X":n,"Y":n}, "Max": {"X":n,"Y":n}}`). Keep it generous but targeted. |

Anchor images should be cropped from a clean, straight scan of the sheet. The scanner binarizes them automatically at load time.

### Config

| Field                 | Type    | Default | Description |
|-----------------------|---------|---------|-------------|
| `blurSize`            | int     | —       | Gaussian blur kernel size, must be odd. Increase for noisier scans. |
| `morphCloseSize`      | int     | —       | Morphological close kernel size. Helps fill gaps in pencil marks. |
| `minAnchorConfidence` | float32 | —       | Minimum template match score (0.0–1.0). Raise if anchors produce false positives. |
| `fillThreshold`       | float64 | 0.5     | Fraction of the inset circle that must be filled for a bubble to count as selected. |
| `bubbleInset`         | float64 | 0.75    | Fraction of the bubble radius sampled when measuring fill. Values below 1.0 exclude the printed border ring from the measurement. |

### Question

| Field          | Type   | Description |
|----------------|--------|-------------|
| `id`           | string | Question identifier used in the output report (e.g. `"Q1"`) |
| `type`         | string | `"single"` (default) flags multiple selections; `"multi"` allows them |
| `bubbleWidth`  | int    | Width of every bubble in this question, in output-space pixels |
| `bubbleHeight` | int    | Height of every bubble in this question, in output-space pixels |
| `options`      | array  | Ordered list of bubbles (see below) |

### Bubble

Bubble coordinates are the top-left corner of the bubble rectangle in output-space pixels. Width and height are shared across all bubbles in a question and defined there instead.

| Field   | Type   | Description |
|---------|--------|-------------|
| `label` | string | Answer label shown in the output report (e.g. `"A"`) |
| `x`     | int    | Left edge of the bubble in output-space pixels |
| `y`     | int    | Top edge of the bubble in output-space pixels |
