#ifndef MAGICK_H
#define MAGICK_H

#include <stddef.h>

#ifdef __cplusplus
extern "C" {
#endif

// Represents a return status for the PDF rendering functions.
typedef enum {
    PDF_OK = 0,           // The rendering was successful.
    PDF_LOADING_ERROR,    // The PDF file was not loaded properly.
    PDF_STARTUP_ERROR,    // There was an error initializing.
    PDF_PAGE_NOT_FOUND,   // The requested page was out of the PDF's bounds.
    PDF_RENDER_ERROR,     // There was some rendering error.
    PDF_OUT_OF_MEMORY,    // The rendering failed because a malloc failed.
    PDF_EXPORT_ERROR,     // The rendered PDF wasn't exported properly.
    PDF_WRONG_PAGE_COUNT, // The number of pages 
} PdfStatus;

//
// Since CGo cannot call C++ directly, we conceal the cv::Mat and cv::Mat[] 
// pointers with void*.
//

// Represents an ordered collection of OpenCV matrices.
typedef struct Mats {
    void* mats;    // cv::Mat[]
    size_t length;
} Mats;

Mats mats_create(size_t length);
void mats_destroy(struct Mats mats);
void* mats_get(struct Mats mats, size_t index);

// Renders all pages in a PDF byte buffer into grayscale OpenCV matrices.
//
// Parameters:
//   buf      - the byte buffer for the PDF
//   density  - the DPI of the rendered PDF
//   status   - the terminal state of the procedure
Mats pdf_bytes_to_mats(
    const void* bytes,
    const size_t n_bytes,
    int density,
    PdfStatus* status
);

#ifdef __cplusplus
}
#endif

#endif
