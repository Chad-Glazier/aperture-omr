CREATE TABLE IF NOT EXISTS marking_templates (
    id TEXT PRIMARY KEY,
    json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS preprocessing_templates (
    id TEXT PRIMARY KEY,
    json TEXT NOT NULL
)
