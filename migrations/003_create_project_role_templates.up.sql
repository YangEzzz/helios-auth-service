-- 创建项目角色模板表
CREATE TABLE project_role_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    role_name VARCHAR(50) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(project_id, role_name)
);

-- 创建索引
CREATE INDEX idx_project_role_templates_project_id ON project_role_templates(project_id);

-- 更新时间触发器
CREATE TRIGGER update_project_role_templates_updated_at 
    BEFORE UPDATE ON project_role_templates 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
