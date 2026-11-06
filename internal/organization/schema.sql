CREATE TABLE IF NOT EXISTS departments (
 id uuid PRIMARY KEY,
 parent_id uuid REFERENCES departments(id) ON DELETE RESTRICT,
 code varchar(64) NOT NULL UNIQUE,
 name varchar(100) NOT NULL,
 sort_order integer NOT NULL DEFAULT 0,
 status varchar(16) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
 version integer NOT NULL DEFAULT 1,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 CHECK (parent_id IS NULL OR parent_id <> id)
);
CREATE UNIQUE INDEX IF NOT EXISTS departments_sibling_name ON departments(COALESCE(parent_id, '00000000-0000-0000-0000-000000000000'::uuid), name);
CREATE INDEX IF NOT EXISTS departments_parent ON departments(parent_id);
CREATE TABLE IF NOT EXISTS department_memberships (
 department_id uuid NOT NULL REFERENCES departments(id) ON DELETE RESTRICT,
 user_id uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 is_manager boolean NOT NULL DEFAULT false,
 created_at timestamptz NOT NULL DEFAULT now(),
 updated_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(department_id, user_id)
);
CREATE INDEX IF NOT EXISTS department_memberships_user ON department_memberships(user_id);
