# Test Output Data

Some tests are difficult to verify automatically. E.g., our binarization function. It's straightforward to check that it handles various inputs without error, and that the output has the correct dimensions, matrix types, etc. However, none of those qualities ensure that the binarized image "came out right." Because of this, many of the automated tests in this package (`omr`) will write visual output to this directory. That way they can be visually inspected for errors.

Some tests will simply draw their output; others will draw a side-by-side image of their input (left) and their output (right). The images themselves contain no text; if you want to check what the output is *supposed* to be, you should refer to the test's source code and the function it's testing for context. Each output file will be prefixed by the name of the test function. 
