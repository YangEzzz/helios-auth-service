-- 删除触发器
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;

-- 删除索引
DROP INDEX IF EXISTS idx_projects_created_at;
DROP INDEX IF EXISTS idx_projects_project_id_string;

-- 删除项目表
DROP TABLE IF EXISTS projects;

