# 数据库设置指南

本项目现在支持PostgreSQL数据库，同时保持向后兼容的内存存储模式。

## 🚀 快速开始

### 1. 使用Docker启动PostgreSQL

```bash
# 启动PostgreSQL容器
docker run --name postgres-fastcopy \
  -e POSTGRES_PASSWORD=yourpassword \
  -e POSTGRES_DB=fastcopy \
  -p 5432:5432 \
  -d postgres:15

# 验证容器运行状态
docker ps
```

### 2. 配置环境变量

复制示例配置文件：
```bash
cp .env.example .env
```

编辑 `.env` 文件，设置数据库连接信息：
```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=yourpassword
DB_NAME=fastcopy
DB_SSLMODE=disable
```

### 3. 运行数据库迁移

```bash
# 方式1: 使用Go脚本
go run scripts/migrate.go -direction=up

# 方式2: 使用Shell脚本 (Linux/Mac)
chmod +x scripts/migrate.sh
./scripts/migrate.sh up
```

### 4. 启动应用

```bash
go run cmd/api/main.go
```

## 📋 配置选项

### 环境变量说明

| 变量名 | 说明 | 默认值 | 必需 |
|--------|------|--------|------|
| `DATABASE_URL` | 完整数据库连接字符串 | - | 否 |
| `DB_HOST` | 数据库主机地址 | localhost | 否 |
| `DB_PORT` | 数据库端口 | 5432 | 否 |
| `DB_USER` | 数据库用户名 | postgres | 否 |
| `DB_PASSWORD` | 数据库密码 | - | 否* |
| `DB_NAME` | 数据库名称 | fastcopy | 否 |
| `DB_SSLMODE` | SSL模式 | disable | 否 |

*注：如果不设置数据库密码，系统将使用内存存储

### 配置优先级

1. **DATABASE_URL** (最高优先级)
2. **分离的DB_*参数**
3. **内存存储** (回退选项)

## 🔧 数据库迁移

### 迁移命令

```bash
# 执行所有向上迁移
go run scripts/migrate.go -direction=up

# 执行所有向下迁移
go run scripts/migrate.go -direction=down

# 迁移指定步数
go run scripts/migrate.go -direction=up -steps=1

# 迁移到指定版本
go run scripts/migrate.go -version=1

# 强制设置版本 (谨慎使用)
go run scripts/migrate.go -force=1
```

### 迁移文件结构

```
migrations/
├── 001_create_users_table.up.sql    # 创建用户表
└── 001_create_users_table.down.sql  # 删除用户表
```

## 🏗️ 数据库架构

### users表结构

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 索引

- `idx_users_email`: 邮箱索引 (用于登录查询)
- `idx_users_created_at`: 创建时间索引 (用于排序)

### 触发器

- `update_users_updated_at`: 自动更新 `updated_at` 字段

## 🔄 开发模式

### 内存存储模式

如果不配置数据库连接，系统将自动使用内存存储：

```bash
# 不设置任何数据库环境变量
unset DATABASE_URL DB_PASSWORD
go run cmd/api/main.go
```

### 混合模式

可以在开发过程中灵活切换：

```bash
# 使用数据库
export DB_PASSWORD=yourpassword
go run cmd/api/main.go

# 切换到内存存储
unset DB_PASSWORD
go run cmd/api/main.go
```

## 🚀 生产部署

### 推荐配置

```env
# 使用DATABASE_URL (推荐)
DATABASE_URL=postgres://username:password@host:5432/dbname?sslmode=require

# 或使用分离参数
DB_HOST=your-postgres-host
DB_PORT=5432
DB_USER=your-username
DB_PASSWORD=your-secure-password
DB_NAME=fastcopy
DB_SSLMODE=require
```

### 安全建议

1. **SSL连接**: 生产环境使用 `DB_SSLMODE=require`
2. **强密码**: 使用复杂的数据库密码
3. **网络隔离**: 限制数据库访问来源
4. **定期备份**: 设置自动备份策略

## 🔍 故障排除

### 常见问题

1. **连接失败**: 检查数据库是否运行，端口是否正确
2. **认证失败**: 验证用户名和密码
3. **迁移失败**: 检查数据库权限，确保用户有创建表的权限

### 日志信息

应用启动时会显示当前使用的存储模式：

```
Using PostgreSQL database (via URL)
Using PostgreSQL database  
Using in-memory storage (no database configured)
```

### 健康检查

```bash
# 检查数据库连接
curl http://localhost:8080/health  # (需要实现健康检查端点)
```

## 📚 扩展功能

项目已预留扩展接口，可轻松添加：

- 用户管理功能 (GetAllUsers, UpdateUser, DeleteUser)
- 分页查询支持
- 软删除功能
- 审计日志
- 数据库连接池监控
