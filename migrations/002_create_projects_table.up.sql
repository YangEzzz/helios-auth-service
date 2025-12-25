-- 创建项目表
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    project_name VARCHAR(255) NOT NULL,
    project_id_string VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    role_documentation TEXT[],  -- PostgreSQL 数组类型
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 创建索引
CREATE INDEX idx_projects_project_id_string ON projects(project_id_string);
CREATE INDEX idx_projects_created_at ON projects(created_at);

-- 为项目表创建更新时间触发器
CREATE TRIGGER update_projects_updated_at 
    BEFORE UPDATE ON projects 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

