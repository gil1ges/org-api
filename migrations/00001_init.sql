-- +goose Up
CREATE TABLE departments (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    parent_id BIGINT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_departments_parent
        FOREIGN KEY (parent_id)
            REFERENCES departments(id)
            ON DELETE CASCADE,
    CONSTRAINT chk_departments_name_nonempty
        CHECK (char_length(btrim(name)) BETWEEN 1 AND 200)
);

CREATE UNIQUE INDEX departments_parent_name_unique_idx
    ON departments (COALESCE(parent_id, 0), lower(name));

CREATE TABLE employees (
    id BIGSERIAL PRIMARY KEY,
    department_id BIGINT NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    position VARCHAR(200) NOT NULL,
    hired_at DATE NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_employees_department
        FOREIGN KEY (department_id)
            REFERENCES departments(id)
            ON DELETE CASCADE,
    CONSTRAINT chk_employees_full_name_nonempty
        CHECK (char_length(btrim(full_name)) BETWEEN 1 AND 200),
    CONSTRAINT chk_employees_position_nonempty
        CHECK (char_length(btrim(position)) BETWEEN 1 AND 200)
);

CREATE INDEX employees_department_id_idx ON employees (department_id);

-- +goose Down
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS departments;
