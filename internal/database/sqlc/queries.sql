-- name: CreateMarkingTemplate :exec
INSERT INTO marking_templates (
    id,
    json
)
VALUES (?, ?);

-- name: CreatePreprocessingTemplate :exec
INSERT INTO preprocessing_templates (
    id,
    json
)
VALUES (?, ?)
