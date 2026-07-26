CREATE TABLE IF NOT EXISTS pdf_render_costs (
    entry_id INTEGER PRIMARY KEY,
    sampled_at INTEGER NOT NULL DEFAULT (strftime('%s', 'now') * 1000),
    pdf_render_baseline INT NOT NULL,
    pdf_render_increment INT NOT NULL
);
