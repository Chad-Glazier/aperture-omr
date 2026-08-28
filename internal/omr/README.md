# OMR

This package contains the core procedures for taking scans and producing marks. The OMR workflow we use is fairly simple, and therefore performant, but is also highly configurable and robust. The main steps are listed below; If you're looking for more detail, I would encourage you to read the (thoroughly-commented) source code yourself.

1) The preprocessing procedure receives a set of pages and attempts to locate specific anchors on each one. The anchors and the information needed to help locate them are included in the preprocessing template. 

>The image below shows a rotated, noisy scan (left) and the locations where the anchors were detected by the preprocessing step (right).

<div align="center">
<img src=".\testdata\output\TestFindAnchors_maxrotation_noisy.png" width="600px" />
</div>

2) Once the anchors are found, a robust method is used to estimate the transformation that would correct each input page. The transformations are applied, which *should* mean that the pages are corrected to match the shape defined by the preprocessing template. 

>The image below shows the input scan (left) and the corrected output of the preprocessing procedure (right).

<div align="center">
<img src=".\testdata\output\TestPreprocess_-5deg_noisy.png" width="600px" />
</div>

3) The marking procedure then binarizes the pages (i.e., converts each pixel to either fully-white or fully-black), and then counts the number of filled pixels in each bubble to produce a fill ratio for each one. The locations of the bubbles are specified by the marking template. The fill ratios are normalized and then used to determine which bubbles are marked and which are not. The output of the marking procedure is a mapping of questions to their marked bubbles and the normalized fill ratio of each bubble.

>The image below shows the binarized scan input (left) and what the marker actually looks at (right).

<div align="center">
<img src=".\testdata\output\TestMarkTemplateMask_preprocessed.png" width="600px" />
</div>

And that's it! We've gone from a low-quality scan to proper marks. The most difficult part is, of course, configuring the system to understand your specific exams. There are two configuration structs that are used to inform the entire process: the preprocessing template, and the marking template. If you're only interested in marking one or two specific kinds of exams (e.g., a standardized 100- or 200-question exam), then it's not too much work to create them by hand. 

Alternatively, if you're integrating Aperture with some kind of exam generation program that can have a variable layout/bubble count, it's strongly recommended that you handle the template generation programmatically. I would be happy to implement more tooling to help create the templates, but I've already sunk enough unpaid labour into this project lol.