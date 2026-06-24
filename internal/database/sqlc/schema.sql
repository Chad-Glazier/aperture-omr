
-- ── Users ────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ── Courses ───────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS courses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    semester TEXT NOT NULL
);

-- ── Enrolments ────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS enrolments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    course_id UUID NOT NULL REFERENCES courses(id),
    role TEXT NOT NULL
);

-- ── Exams ─────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS exams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    course_id UUID NOT NULL REFERENCES courses(id),
    created_by UUID NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    version_label TEXT,
    exam_date TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- ── Exam Templates ────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS exam_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id),
    question_config JSONB NOT NULL,
    answer_key JSONB NOT NULL,
    scoring_rules JSONB,
    total_questions INT NOT NULL,
    version INT DEFAULT 1,
    omr_template_id UUID REFERENCES omr_templates(id),
    generated_pdf_path TEXT
);

-- ── Scans ─────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS scans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_id UUID NOT NULL REFERENCES exams(id),
    uploader_user_id UUID NOT NULL REFERENCES users(id),
    path TEXT NOT NULL,
    page_count INT
);

-- ── OMR Templates ─────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS omr_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    page_count INT NOT NULL,
    version INT DEFAULT 1,
    created_at TIMESTAMPTZ DEFAULT now(),
    scan_template_path TEXT,
    mark_template_path TEXT
);

-- ── OMR Markings ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS omr_markings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_id UUID NOT NULL REFERENCES scans(id),
    omr_template_id UUID NOT NULL REFERENCES omr_templates(id),
    status TEXT DEFAULT 'pending',
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

-- ── OMR Detected Marks ────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS omr_detected_marks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marking_id UUID NOT NULL REFERENCES omr_markings(id),
    question_number INT NOT NULL,
    detected_value TEXT,
    confidence_score REAL,
    needs_review BOOLEAN DEFAULT false,
    x INT,
    y INT,
    width INT,
    height INT
);

-- ── Exam Grades ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS exam_grades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    omr_marking_id UUID UNIQUE NOT NULL REFERENCES omr_markings(id),
    exam_template_id UUID NOT NULL REFERENCES exam_templates(id),
    question_results JSONB NOT NULL,
    total_score NUMERIC,
    max_score NUMERIC,
    template_version INT,
    graded_at TIMESTAMPTZ DEFAULT now()
);

-- ── Review Requests ───────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS review_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    exam_grade_id UUID NOT NULL REFERENCES exam_grades(id),
    student_id UUID NOT NULL REFERENCES users(id),
    question_number TEXT,
    student_comment TEXT,
    status TEXT DEFAULT 'pending',
    instructor_response TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    resolved_at TIMESTAMPTZ
);
