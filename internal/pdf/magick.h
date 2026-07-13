#ifndef MAGICK_H
#define MAGICK_H

#include <stddef.h>

// Gets the number of pages in a PDF file (without rendering it). Returns -1 if
// the file could not be loaded.
int pdf_file_page_count(const char *file_name);

// Represents a return status for the PDF rendering functions.
typedef enum {
    PDF_OK = 0,         // The rendering was successful.
    PDF_LOADING_ERROR,  // The PDF file was not loaded properly.
    PDF_PAGE_NOT_FOUND, // The requested page was out of the PDF's bounds.
    PDF_RENDER_ERROR,   // There was some rendering error.
    PDF_OUT_OF_MEMORY,  // The rendering failed because a malloc failed.
    PDF_EXPORT_ERROR,   // The rendered PDF wasn't exported properly.
} PdfStatus;

// Represents an image in grayscale.
typedef struct {
    unsigned char *pixels;
    size_t width;
    size_t height;
} GrayImage;

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
PdfStatus pdf_file_page_to_gray(
    const char *file_name,
    int density,
    int page_idx,
    GrayImage *result
);

// Frees an allocated GrayImage.
void free_gray_image(GrayImage *img);

#endif
