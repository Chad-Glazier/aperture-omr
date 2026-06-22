-- work in progress: we need to store preprocessing template data in the DB,
-- and the anchor images must be stored in the image store. Full marking
-- templates should also be stored in the database, though we have no idea what
-- those will look like yet. Records of uploaded scans may also be worth 
-- storing, though we might instead be able to collect that metadata from S3
-- and have no need for separate records.

CREATE TABLE IF NOT EXISTS omr_preprocessing_templates (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    image_width INT NOT NULL,
    image_height INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS omr_preprocessing_template_configs (
    preprocessing_template_id BIGINT PRIMARY KEY,
    blur_size INT NOT NULL,
    morph_close_size INT NOT NULL,
    min_anchor_confidence DECIMAL(4, 3) NOT NULL,
    CONSTRAINT fk_preprocessing_template_config FOREIGN KEY (preprocessing_template_id) REFERENCES omr_preprocessing_templates(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS omr_preprocessing_template_anchors (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    preprocessing_template_id BIGINT NOT NULL,
    anchor_path VARCHAR(500) NOT NULL,
    center_x INT NOT NULL,
    center_y INT NOT NULL,
    roi_min_x INT NOT NULL,
    roi_min_y INT NOT NULL,
    roi_max_x INT NOT NULL,
    roi_max_y INT NOT NULL,
    CONSTRAINT fk_preprocessing_template_anchor 
        FOREIGN KEY (preprocessing_template_id) 
        REFERENCES omr_preprocessing_templates(id) 
        ON DELETE CASCADE
);
