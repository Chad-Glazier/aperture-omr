#include "magick.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <assert.h>

#include <wand/MagickWand.h>

// 
// Types.
//

GrayImage* gray_image_create(size_t width, size_t height, void* pixels) {

    GrayImage* img = malloc(sizeof(GrayImage));
    if (img == NULL) {
        return NULL;
    }

    img->width = width;
    img->height = height;
    img->pixels = pixels;

    return img;
}

void gray_image_destroy(GrayImage* img) {
    if (img == NULL) {
        return;
    }

    if (img->pixels != NULL) {
        free(img->pixels);
    }

    free(img);
}

GrayImageSlice* gray_image_slice_create(size_t length) {

    GrayImageSlice* slice = malloc(sizeof(GrayImageSlice));
    if (slice == NULL) {
        return NULL;
    }

    slice->items = calloc(length, sizeof(GrayImage));
    if (slice->items == NULL) {
        free(slice);
        return NULL;
    }

    slice->length = length;

    return slice;
}

void gray_image_slice_destroy(GrayImageSlice* slice) {
    if (slice == NULL) {
        return;
    }

    if (slice->items != NULL) {
        free(slice->items);
    }

    free(slice);
}

//
// Get the number of pages in a PDF file.
//

int pdf_file_page_count(const char* file_name) {
    MagickWand* wand = NewMagickWand();

    if (MagickPingImage(wand, file_name) == MagickFalse) {
        return -1;
    }

    size_t pages = MagickGetNumberImages(wand);

    return pages;
}

//
// Convert a single page from a PDF file to a grayscale image.
//

GrayImage* pdf_file_page_to_gray(
    const char* file_name,
    size_t density,
    size_t page_idx,
    PdfStatus* status
) {
    *status = PDF_OK;

    MagickWand* wand = NewMagickWand(); // <https://youtu.be/mdCyzJT59nw?si=gp8CU9qG2sbTp081>
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
        char* msg = MagickGetException(wand, &severity);

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

        *status = PDF_PAGE_NOT_FOUND;
        return NULL;
    }

    char* format = MagickGetImageFormat(wand);
    if (strcmp(format, "PDF") != 0) {

        MagickRelinquishMemory(format);
        DestroyMagickWand(wand);

        *status = PDF_LOADING_ERROR;
        return NULL;
    }
    MagickRelinquishMemory(format);

    assert(MagickGetNumberImages(wand) == 1);

    size_t width = MagickGetImageWidth(wand);
    size_t height = MagickGetImageHeight(wand);

    void* pixels = malloc(width * height);
    if (pixels == NULL) {

        DestroyMagickWand(wand);

        *status = PDF_OUT_OF_MEMORY;
        return NULL;
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

        *status = PDF_EXPORT_ERROR;
        return NULL;
    }

    GrayImage* image = gray_image_create(width, height, pixels);
    if (image == NULL) {

        free(pixels);
        DestroyMagickWand(wand);

        *status = PDF_OUT_OF_MEMORY;
        return NULL;
    }

    DestroyMagickWand(wand);

    return image;
}

//
// Convert all pages in a PDF file into grayscale images.
//

GrayImageSlice* pdf_file_to_gray_images(
    const char* file_name,
    size_t density,
    PdfStatus* status
) {
    *status = PDF_OK;

    MagickWand *wand = NewMagickWand();
    if (wand == NULL) {

        *status = PDF_STARTUP_ERROR;
        return NULL;
    }

    MagickSetResolution(wand, density, density);

    if (MagickReadImage(wand, file_name) == MagickFalse) {
        
        DestroyMagickWand(wand);
        
        *status = PDF_LOADING_ERROR;
        return NULL;
    }

    size_t page_count = MagickGetNumberImages(wand);

    GrayImageSlice *slice = gray_image_slice_create(page_count);
    if (slice == NULL) {
        
        DestroyMagickWand(wand);
        
        *status = PDF_OUT_OF_MEMORY;
        return NULL;
    }

    MagickResetIterator(wand);

    size_t page = 0;

    while (MagickNextImage(wand) != MagickFalse) {
        if (MagickSetImageColorspace(wand, GRAYColorspace) == MagickFalse ||
            MagickSetImageType(wand, GrayscaleType) == MagickFalse) {

            gray_image_slice_destroy(slice);
            DestroyMagickWand(wand);

            *status = PDF_RENDER_ERROR;
            return NULL;
        }

        size_t width = MagickGetImageWidth(wand);
        size_t height = MagickGetImageHeight(wand);
        size_t bytes = width * height;

        unsigned char *pixels = malloc(bytes);
        if (pixels == NULL) {

            gray_image_slice_destroy(slice);
            DestroyMagickWand(wand);

            *status = PDF_OUT_OF_MEMORY;
            return NULL;
        }

        if (MagickExportImagePixels(
                wand,
                0,
                0,
                width,
                height,
                "I",
                CharPixel,
                pixels) == MagickFalse) {

            free(pixels);
            gray_image_slice_destroy(slice);
            DestroyMagickWand(wand);

            *status = PDF_EXPORT_ERROR;
            return NULL;
        }

        slice->items[page].pixels = pixels;
        slice->items[page].width = width;
        slice->items[page].height = height;

        page++;
    }

    DestroyMagickWand(wand);

    return slice;
}

//
// Convert PDF bytes into grayscale images.
//

GrayImageSlice* pdf_bytes_to_gray_images(
    const void* bytes,
    const size_t n_bytes,
    int density,
    PdfStatus* status
) {
    *status = PDF_OK;

    MagickWand* wand = NewMagickWand();
    MagickSetResolution(wand, density, density);

    if (MagickReadImageBlob(wand, bytes, n_bytes) == MagickFalse) {

        DestroyMagickWand(wand);

        *status = PDF_LOADING_ERROR;
        return NULL;
    }

    char* format = MagickGetImageFormat(wand);
    if (strcmp(format, "PDF") != 0) {

        MagickRelinquishMemory(format);
        DestroyMagickWand(wand);

        *status = PDF_LOADING_ERROR;
        return NULL;
    }
    MagickRelinquishMemory(format);

    GrayImageSlice* images =
        gray_image_slice_create(MagickGetNumberImages(wand));
    if (images == NULL) {

        DestroyMagickWand(wand);

        *status = PDF_OUT_OF_MEMORY;
        return NULL;
    }

    MagickResetIterator(wand);
    while (MagickNextImage(wand) != MagickFalse) {

        MagickSetImageColorspace(wand, GRAYColorspace);
        MagickSetImageType(wand, GrayscaleType);

        const size_t width = MagickGetImageWidth(wand);
        const size_t height = MagickGetImageHeight(wand);

        unsigned char* pixels = malloc(width * height);
        if (pixels == NULL) {

            gray_image_slice_destroy(images);
            DestroyMagickWand(wand);

            *status = PDF_OUT_OF_MEMORY;
            return NULL;
        }

        if (MagickExportImagePixels(
            wand,
            0,
            0,
            width,
            height,
            "I",
            CharPixel,
            pixels) == MagickFalse) {

            free(pixels);
            gray_image_slice_destroy(images);
            DestroyMagickWand(wand);

            *status = PDF_RENDER_ERROR;
            return NULL;
        }

        int i = MagickGetIteratorIndex(wand);
        images->items[i].pixels = pixels;
        images->items[i].width = width;
        images->items[i].height = height;

    }

    DestroyMagickWand(wand);
    return images;
}
