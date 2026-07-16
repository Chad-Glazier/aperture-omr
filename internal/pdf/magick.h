#ifndef MAGICK_H
#define MAGICK_H

#include <stddef.h>

// Gets the number of pages in a PDF file (without rendering it). Returns -1 if
// the file could not be loaded.
int pdf_file_page_count(const char* file_name);

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

// Represents an image in grayscale.
typedef struct {
    void* pixels;
    size_t width;
    size_t height;
} GrayImage;

GrayImage* gray_image_create(size_t width, size_t height, void* pixels);
void gray_image_destroy(GrayImage* img);

// Represents a slice of images in grayscale.
typedef struct {
    size_t length;
    GrayImage* items;
} GrayImageSlice;

GrayImageSlice* gray_image_slice_create(size_t length);
void gray_image_slice_destroy(GrayImageSlice* slice);

// Renders a single page from a PDF file to a grayscale image.
//
// Parameters:
//   file_name - the name of a PDF file. The filename must have the ".pdf" 
//               extension
//   density   - the DPI of the rendered PDF
//   page_idx  - zero-based index for the page
//   result    - the object where the result will be stored
//
// Returns:
//   PDF_OK             - success 
//   PDF_PAGE_NOT_FOUND - the page index was out of bounds
//   otherwise          - the function failed for another reason.
GrayImage* pdf_file_page_to_gray(
    const char* file_name,
    size_t density,
    size_t page_idx,
    PdfStatus* status
);

// Renders all pages in a PDF byte buffer into grayscale images.
//
// Parameters:
//   buf      - the byte buffer for the PDF
//   
//   density  - the DPI of the rendered PDF
//   status   - the terminal state of the procedure
GrayImageSlice* pdf_bytes_to_gray_images(
    const void* bytes,
    const size_t n_bytes,
    int density,
    PdfStatus* status
);

// Renders all pages in a PDF byte buffer into grayscale images.
//
// Parameters:
//   buf      - the byte buffer for the PDF
//   density  - the DPI of the rendered PDF
//   status   - the terminal state of the procedure
GrayImageSlice* pdf_file_to_gray_images(
    const char* file_name,
    size_t density,
    PdfStatus* status
);

#endif
