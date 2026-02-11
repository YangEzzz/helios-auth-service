-- 创建审计日志表
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL, -- 如果用户被删除，日志保留但user_id设为NULL
    action VARCHAR(100) NOT NULL,      -- 操作类型 (e.g., "approve_user", "create_project")
    resource VARCHAR(255),             -- 操作对象 (e.g., "user:uuid-123")
    details TEXT,                      -- 详细信息 (JSON格式字符串)
    ip_address VARCHAR(45),            -- 操作者IP
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
