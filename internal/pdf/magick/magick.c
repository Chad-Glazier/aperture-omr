#include "magick.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <assert.h>

#include <wand/MagickWand.h>

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

        if (strstr(
                msg, 
                "Requested FirstPage is greater than the number of pages"
            ) != NULL) {

            MagickRelinquishMemory(msg);
            DestroyMagickWand(wand);
            return PDF_PAGE_NOT_FOUND;
        }

        MagickRelinquishMemory(msg);
        DestroyMagickWand(wand);

        return PDF_RENDER_ERROR;
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
