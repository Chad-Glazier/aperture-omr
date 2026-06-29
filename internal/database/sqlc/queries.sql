PRAGMA foreign_keys = ON;

-- =========================
-- PREPROCESSING TEMPLATES
-- =========================

CREATE TABLE IF NOT EXISTS preprocessing_templates (
    id TEXT PRIMARY KEY,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    blur_size INTEGER NOT NULL,
    morph_close_size INTEGER NOT NULL,
    min_anchor_confidence REAL NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS preprocessing_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id TEXT NOT NULL,
    page_index INTEGER NOT NULL,
    FOREIGN KEY (template_id) REFERENCES preprocessing_templates(id) ON DELETE CASCADE,
    UNIQUE(template_id, page_index)
);

-- 3 anchors per page
CREATE TABLE IF NOT EXISTS preprocessing_anchors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,

    anchor_index INTEGER NOT NULL,

    center_x INTEGER NOT NULL,
    center_y INTEGER NOT NULL,

    roi_min_x INTEGER NOT NULL,
    roi_min_y INTEGER NOT NULL,
    roi_max_x INTEGER NOT NULL,
    roi_max_y INTEGER NOT NULL,

    image BLOB NOT NULL,

    FOREIGN KEY (page_id) REFERENCES preprocessing_pages(id) ON DELETE CASCADE,
    UNIQUE(page_id, anchor_index)
);

-- =========================
-- SCANS
-- =========================

CREATE TABLE IF NOT EXISTS scans (
    id TEXT PRIMARY KEY,
    preprocessing_template_id TEXT NOT NULL,
    created_at INTEGER NOT NULL,

    FOREIGN KEY (preprocessing_template_id)
        REFERENCES preprocessing_templates(id)
);

-- each page is stored externally via key
CREATE TABLE IF NOT EXISTS scan_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scan_id TEXT NOT NULL,
    page_index INTEGER NOT NULL,
    image_key TEXT NOT NULL,

    FOREIGN KEY (scan_id) REFERENCES scans(id) ON DELETE CASCADE,
    UNIQUE(scan_id, page_index)
);

-- =========================
-- MARKING TEMPLATES
-- =========================

CREATE TABLE IF NOT EXISTS marking_templates (
    id TEXT PRIMARY KEY,
    fill_threshold REAL NOT NULL,
    bubble_inset REAL NOT NULL,
    flag_threshold REAL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS marking_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id TEXT NOT NULL,
    page_index INTEGER NOT NULL,

    FOREIGN KEY (template_id) REFERENCES marking_templates(id) ON DELETE CASCADE,
    UNIQUE(template_id, page_index)
);

CREATE TABLE IF NOT EXISTS marking_questions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    page_id INTEGER NOT NULL,

    question_id TEXT NOT NULL,
    bubble_width INTEGER NOT NULL,
    bubble_height INTEGER NOT NULL,
    question_type TEXT, -- NULL or 'multi'

    FOREIGN KEY (page_id) REFERENCES marking_pages(id) ON DELETE CASCADE,
    UNIQUE(page_id, question_id)
);

CREATE TABLE IF NOT EXISTS marking_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    question_id INTEGER NOT NULL,

    label TEXT NOT NULL,
    x INTEGER NOT NULL,
    y INTEGER NOT NULL,

    FOREIGN KEY (question_id) REFERENCES marking_questions(id) ON DELETE CASCADE,
    UNIQUE(question_id, label)
);

-- =========================
-- MARKING JOBS (async)
-- =========================

CREATE TABLE IF NOT EXISTS marking_jobs (
    id TEXT PRIMARY KEY,
    template_id TEXT NOT NULL,
    status TEXT NOT NULL, 
    -- pending | running | completed | failed

    start_time INTEGER,
    end_time INTEGER,

    FOREIGN KEY (template_id) REFERENCES marking_templates(id)
);

CREATE TABLE IF NOT EXISTS marking_job_scans (
    job_id TEXT NOT NULL,
    scan_id TEXT NOT NULL,

    PRIMARY KEY (job_id, scan_id),

    FOREIGN KEY (job_id) REFERENCES marking_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (scan_id) REFERENCES scans(id)
);

-- =========================
-- MARKING RESULTS
-- =========================

CREATE TABLE IF NOT EXISTS scan_marks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id TEXT NOT NULL,
    scan_id TEXT NOT NULL,

    question_id TEXT NOT NULL,
    flagged INTEGER NOT NULL DEFAULT 0,

    FOREIGN KEY (job_id) REFERENCES marking_jobs(id) ON DELETE CASCADE,
    FOREIGN KEY (scan_id) REFERENCES scans(id)
);

CREATE TABLE IF NOT EXISTS scan_mark_options (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mark_id INTEGER NOT NULL,

    selected_label TEXT NOT NULL,

    FOREIGN KEY (mark_id) REFERENCES scan_marks(id) ON DELETE CASCADE
);
