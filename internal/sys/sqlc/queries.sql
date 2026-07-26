-- name: GetPdfRenderCosts :one
SELECT 
    *
FROM
    pdf_render_costs
ORDER BY
    sampled_at DESC
LIMIT 1;

-- name: CreatePdfRenderCosts :exec
INSERT INTO
    pdf_render_costs (pdf_render_baseline, pdf_render_increment)
VALUES
    (?, ?);
