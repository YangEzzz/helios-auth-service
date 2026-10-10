CREATE TABLE IF NOT EXISTS sso_sessions (
 id varchar(64) PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id),
 expires_at timestamptz NOT NULL, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sso_sessions_expiry ON sso_sessions(expires_at);
CREATE TABLE IF NOT EXISTS sso_clients (
 id varchar(100) PRIMARY KEY, project_id uuid NOT NULL REFERENCES projects(id),
 name varchar(255) NOT NULL, secret_hash varchar(64) NOT NULL,
 redirect_uris text[] NOT NULL, enabled boolean NOT NULL DEFAULT true,
 created_at timestamptz NOT NULL DEFAULT now(), updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS sso_requests (
 id varchar(64) PRIMARY KEY, browser_hash varchar(64) NOT NULL,
 client_id varchar(100) NOT NULL REFERENCES sso_clients(id), redirect_uri text NOT NULL,
 state varchar(128) NOT NULL, challenge varchar(43) NOT NULL,
 expires_at timestamptz NOT NULL, used_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sso_requests_expiry ON sso_requests(expires_at);
CREATE TABLE IF NOT EXISTS sso_codes (
 id varchar(64) PRIMARY KEY, user_id uuid NOT NULL REFERENCES users(id),
 client_id varchar(100) NOT NULL REFERENCES sso_clients(id), redirect_uri text NOT NULL,
 challenge varchar(43) NOT NULL, expires_at timestamptz NOT NULL,
 used_at timestamptz, created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_sso_codes_expiry ON sso_codes(expires_at);
