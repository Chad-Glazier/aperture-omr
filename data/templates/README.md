# OMR Templates

A template describes everything the scanner needs to preprocess a specific bubble sheet design: the output dimensions, anchor points used for perspective correction, and image processing configuration.

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

| Field    | Type | Description |
|----------|------|-------------|
| `width`  | int  | Output image width in pixels after warping |
| `height` | int  | Output image height in pixels after warping |
| `anchors` | array | Exactly 3 anchor definitions (see below) |
| `config` | object | Preprocessing parameters (see below) |

### Anchor

Each anchor is a distinctive region of the sheet used to compute the perspective transform. All coordinates are in output (`width` x `height`) space for consistency. Ideally, choose each anchor to be in a distinct corner of the page for the best results.

| Field    | Type   | Description |
|----------|--------|-------------|
| `path`   | string | Path to a cropped image of the anchor region, relative to the JSON file |
| `center` | Point  | Where the center of this anchor should appear in the output (`{"X": n, "Y": n}`) |
| `roi`    | Rect   | Search region in output space (`{"Min": {"X":n,"Y":n}, "Max": {"X":n,"Y":n}}`). Keep it generous but targeted. |

Anchor images should be cropped from a clean, straight scan of the sheet and then binarized, the scanner binarizes them automatically at load time.

### Config

| Field                 | Type    | Description |
|-----------------------|---------|-------------|
| `blurSize`            | int     | Gaussian blur kernel size, must be odd. Increase for noisier scans. |
| `morphCloseSize`      | int     | Morphological close kernel size. Helps fill gaps in pencil marks. |
| `minAnchorConfidence` | float32 | Minimum template match confidence (0.0–1.0). Raise if anchors produce false positives. |
