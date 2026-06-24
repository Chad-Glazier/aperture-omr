# OMR Templates

A form is described by two JSON files: a **scan template** that tells the preprocessor how to correct perspective, and a **mark template** that tells the marker where each answer bubble is located.

## Usage

```
# Preprocess only (saves binary image for later marking)
omr preprocess <image> <scan-template> --output preprocessed.png

# Mark only (requires a preprocessed binary image)
omr mark <preprocessed.png> <mark-template>

# Preprocess and mark in one step
omr scan <image> <scan-template> <mark-template>
```

## Example Directory Structure

```
my_form/
├── scan.json
├── mark.json
└── anchors/
    ├── logo.jpg
    ├── footer.jpg
    └── info.jpg
```

Anchor image paths in the scan template are resolved relative to the JSON file's location.

---

## Scan Template

Controls perspective correction and binarization. Pass this to `preprocess` or `scan`.

### Top-level fields

| Field     | Type   | Description |
|-----------|--------|-------------|
| `width`   | int    | Output image width in pixels after warping |
| `height`  | int    | Output image height in pixels after warping |
| `anchors` | array  | Exactly 3 anchor definitions (see below) |
| `config`  | object | Preprocessing parameters (see below) |

### Anchor

Each anchor is a distinctive region of the sheet used to compute the affine perspective transform. All coordinates are in output (`width` × `height`) space. Choose anchors in distinct corners of the page for the most accurate warp.

| Field    | Type   | Description |
|----------|--------|-------------|
| `path`   | string | Path to a cropped image of the anchor region, relative to the JSON file |
| `center` | Point  | Where the centre of this anchor should land in the output (`{"X": n, "Y": n}`) |
| `roi`    | Rect   | Search region in output space (`{"Min": {"X":n,"Y":n}, "Max": {"X":n,"Y":n}}`). Keep it generous but targeted. |

Anchor images should be cropped from a clean, straight scan of the sheet. They are binarized automatically at load time.

### Scan Config

| Field                 | Type    | Description |
|-----------------------|---------|-------------|
| `blurSize`            | int     | Gaussian blur kernel size, must be odd. Increase for noisier scans. |
| `morphCloseSize`      | int     | Morphological close kernel size. Helps fill gaps in pencil marks. |
| `minAnchorConfidence` | float32 | Minimum template-match score (0.0–1.0). Raise if anchors produce false positives. |

---

## Mark Template

Defines question and bubble locations. Pass this to `mark` or `scan`.

### Top-level fields

| Field       | Type   | Description |
|-------------|--------|-------------|
| `config`    | object | Marking parameters (see below) |
| `questions` | array  | Ordered list of questions and their bubble locations (see below) |

### Mark Config

| Field           | Type    | Default | Description |
|-----------------|---------|---------|-------------|
| `fillThreshold` | float64 | 0.5     | Minimum fill ratio for a bubble to count as selected (0.0–1.0). |
| `bubbleInset`   | float64 | 0.75    | Fraction of the bubble radius sampled. Values below 1.0 exclude the printed border ring from the measurement. |

### Question

| Field          | Type   | Description |
|----------------|--------|-------------|
| `id`           | string | Identifier used in the output report (e.g. `"Q1"`) |
| `type`         | string | `"single"` (default) flags multiple selections; `"multi"` allows them |
| `bubbleWidth`  | int    | Width of every bubble in this question, in output-space pixels |
| `bubbleHeight` | int    | Height of every bubble in this question, in output-space pixels |
| `options`      | array  | Ordered list of bubbles (see below) |

### Bubble

Coordinates are the top-left corner of the bubble rectangle in output-space pixels. Dimensions are shared across all bubbles in a question and defined at the question level.

| Field   | Type   | Description |
|---------|--------|-------------|
| `label` | string | Answer label shown in the output report (e.g. `"A"`) |
| `x`     | int    | Left edge of the bubble in output-space pixels |
| `y`     | int    | Top edge of the bubble in output-space pixels |
