#include "magick.h"

#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <stdint.h>
#include <assert.h>

#include <wand/MagickWand.h>
#include <opencv2/opencv.hpp>

// 
// Types.
//

Mats mats_create(size_t length) {
    Mats mats;
    mats.length = length;
    mats.mats = new cv::Mat[length];

    return mats;
}

void mats_destroy(struct Mats mats) {
    delete[] static_cast<cv::Mat*>(mats.mats);
}

void* mats_get(struct Mats mats, size_t index) {
    if (index >= mats.length) {
        return nullptr;
    }

    return &static_cast<cv::Mat*>(mats.mats)[index];
}

//
// Convert PDF bytes into grayscale OpenCV matrices.
//

Mats pdf_bytes_to_mats(
    const void* bytes,
    const size_t n_bytes,
    int density,
    PdfStatus* status
) {
    *status = PDF_OK;

    MagickWand* wand = NewMagickWand(); // <https://youtu.be/mdCyzJT59nw?si=w86B_1-z85RDTNhL>
    MagickSetResolution(wand, density, density);

    if (MagickReadImageBlob(wand, bytes, n_bytes) == MagickFalse) {

        DestroyMagickWand(wand);

        *status = PDF_LOADING_ERROR;
        return Mats{nullptr, 0};
    }

    char* format = MagickGetImageFormat(wand);
    if (strcmp(format, "PDF") != 0) {

        MagickRelinquishMemory(format);
        DestroyMagickWand(wand);

        *status = PDF_LOADING_ERROR;
        return Mats{nullptr, 0};
    }
    MagickRelinquishMemory(format);

    Mats mats = mats_create(MagickGetNumberImages(wand));

    size_t i = 0;
    MagickResetIterator(wand);
    while (MagickNextImage(wand) != MagickFalse) {

        const size_t width = MagickGetImageWidth(wand);
        const size_t height = MagickGetImageHeight(wand);

        cv::Mat& mat = static_cast<cv::Mat*>(mats.mats)[i++];
        mat = cv::Mat(
            static_cast<int>(height),
            static_cast<int>(width),
            CV_8UC1
        );

        MagickSetImageColorspace(wand, GRAYColorspace);
        MagickSetImageType(wand, GrayscaleType);

        if (MagickExportImagePixels(
            wand,
            0,
            0,
            width,
            height,
            "I",
            CharPixel,
            mat.data) == MagickFalse) {

            mats_destroy(mats);
            DestroyMagickWand(wand);

            *status = PDF_RENDER_ERROR;
            return Mats{nullptr, 0};
        }

        MagickRemoveImage(wand);
        MagickResetIterator(wand);
    }

    DestroyMagickWand(wand);
    return mats;
}
