-- name: CreateMarkingTemplate :exec
INSERT INTO
    marking_templates (id, bytes)
VALUES
    (?, ?);

-- name: GetMarkingTemplate :one
SELECT
    *
FROM
    marking_templates
WHERE
    id = ?;

-- name: DeleteMarkingTemplate :execrows
DELETE FROM
    marking_templates
WHERE
    id = ?;

-- name: CreatePreprocessingTemplate :exec
INSERT INTO
    preprocessing_templates (id, bytes)
VALUES
    (?, ?);

-- name: GetPreprocessingTemplate :one
SELECT
    *
FROM
    preprocessing_templates
WHERE
    id = ?;

-- name: DeletePreprocessingTemplate :execrows
DELETE FROM
    preprocessing_templates
WHERE
    id = ?;

-- name: CreateScan :exec
INSERT INTO
    scans (id, preprocessing_template_id, created_at_unix_ms)
VALUES
    (?, ?, ?);

-- name: CreateScanPage :exec
INSERT INTO
    scan_pages (matrix_key, picture_key, page_index, scan_id)
VALUES
    (?, ?, ?, ?);

-- name: CountScans :one
SELECT
    count(*)
FROM
    scans;

-- name: DeleteScan :execrows
DELETE FROM
    scans
WHERE
    id = ?;

-- name: GetScansFromBefore :many
SELECT
    *
FROM
    scans
WHERE
    created_at_unix_ms < ?;

-- name: GetScanPage :one
SELECT 
    *
FROM
    scan_pages
WHERE
    scan_id = ? AND
    page_index = ?
LIMIT 1;

-- name: GetPagesForScan :many
SELECT 
    *
FROM
    scan_pages
WHERE
    scan_id = ?
ORDER BY
    page_index ASC;

-- name: DeletePagesForScan :execrows
DELETE FROM
    scan_pages
WHERE
    scan_id = ?;
