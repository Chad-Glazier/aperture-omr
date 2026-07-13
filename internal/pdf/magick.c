#include "magick.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <assert.h>

#include <wand/MagickWand.h>

//
// Get the number of pages in a PDF file.
//

int pdf_file_page_count(const char *file_name) {
    MagickWand *wand = NewMagickWand();

    if (MagickPingImage(wand, file_name) == MagickFalse) {
        return -1;
    }

    size_t pages = MagickGetNumberImages(wand);

    return pages;
}

//
// Convert a single page from a PDF file to a grayscale image.
//

PdfStatus pdf_file_page_to_gray(
    const char *file_name,
    int density,
    int page_idx,
    GrayImage *result
) {
    assert(page_idx >= 0);
    assert(density > 0);

    memset(result, 0, sizeof(*result));

    MagickWand *wand = NewMagickWand();
    MagickSetResolution(wand, density, density);

    char spec[4096];
    snprintf(
        spec,
        sizeof(spec),
        "%s[%d]",
        file_name,
        page_idx
    );

    if (MagickReadImage(wand, spec) == MagickFalse) {
        ExceptionType severity;
        char *msg = MagickGetException(wand, &severity);

        //
        // Previously, we had an exception handler to check whether the error 
        // is actually an out-of-bounds index. However, that doesn't work on 
        // Linux because the delegated call to ghostscript does not yield 
        // capturable output, instead it just prints it right to stderr. This 
        // leads to the approach of simply assuming that any failure in loading
        // the PDF is an index out-of-bounds error. This isn't ideal, but it 
        // works as long as we dont get a malformed PDF. (Notably, a malformed 
        // PDF can still be detected if the total number of pages rendered is 
        // zero.)
        //
        // We could use the `pdf_get_page_count` function, but the cost of 
        // getting metadata from the PDF is still significant--in my testing,
        // about 2 full seconds for a large PDF. We should look into writing
        // our own page counter in the future if the issue becomes important.
        //

        MagickRelinquishMemory(msg);
        DestroyMagickWand(wand);

        return PDF_PAGE_NOT_FOUND;
    }

    char *format = MagickGetImageFormat(wand);
    if (strcmp(format, "PDF") != 0) {

        MagickRelinquishMemory(format);
        DestroyMagickWand(wand);

        return PDF_LOADING_ERROR;
    }

    assert(MagickGetNumberImages(wand) == 1);

    size_t width = MagickGetImageWidth(wand);
    size_t height = MagickGetImageHeight(wand);

    unsigned char *pixels = malloc(width * height);
    if (pixels == NULL) {
        DestroyMagickWand(wand);
        return PDF_OUT_OF_MEMORY;
    }

    MagickSetImageColorspace(wand, GRAYColorspace);

    if (MagickExportImagePixels(
            wand,
            0,
            0,
            width,
            height,
            "I",
            CharPixel,
            pixels
        ) == MagickFalse) {

        free(pixels);
        DestroyMagickWand(wand);
        return PDF_EXPORT_ERROR;
    }

    result->pixels = pixels;
    result->width = width;
    result->height = height;

    DestroyMagickWand(wand);

    return PDF_OK;
}

void free_gray_image(GrayImage *img) {
    if (img->pixels != NULL) {
        free(img->pixels);
    }

    img->pixels = NULL;
    img->width = 0;
    img->height = 0;
}
