# helios-auth-service

这是一个为“速贴”项目设计的Go后端服务，支持用户注册和登录功能，同时支持PostgreSQL数据库和内存存储两种模式。

## 结构介绍

请参考 `intro.md` 文件获取详细的项目结构和文件职责说明。

## 🚀 快速开始

### 方式一：使用内存存储（快速体验）

1.  **安装依赖：** 确保您已安装Go环境。
2.  **下载依赖：** `go mod tidy`
3.  **创建 `.env` 文件：**
    ```bash
    cp .env.example .env-production
    ```
    编辑 `.env` 文件，至少设置：
    ```
    PORT=8080
    JWT_SECRET=your_super_secret_key_here
    ```
4.  **运行服务：** `go run cmd/api/main.go`

### 方式二：使用PostgreSQL数据库（推荐）

1.  **启动PostgreSQL：**
    ```bash
    docker run --name postgres-fastcopy \
      -e POSTGRES_PASSWORD=yourpassword \
      -e POSTGRES_DB=fastcopy \
      -p 5432:5432 \
      -d postgres:15
    ```

2.  **配置数据库连接：** 编辑 `.env` 文件：
    ```
    PORT=8080
    JWT_SECRET=your_super_secret_key_here
    DB_HOST=localhost
    DB_PORT=5432
    DB_USER=postgres
    DB_PASSWORD=yourpassword
    DB_NAME=fastcopy
    ```

3.  **运行数据库迁移：**
    ```bash
    go run scripts/migrate.go -direction=up
    ```

4.  **启动服务：** `go run cmd/api/main.go`

服务将在 `http://localhost:8080` 启动。

## API 接口

*   **注册：** `POST /api/auth/register`
    *   请求体示例：`{"email": "test@example.com", "password": "password123"}`
*   **登录：** `POST /api/auth/login`
    *   请求体示例：`{"email": "test@example.com", "password": "password123"}`

## 📋 功能特性

*   **双存储模式**: 支持PostgreSQL数据库和内存存储
*   **自动回退**: 数据库连接失败时自动使用内存存储
*   **数据库迁移**: 内置迁移工具，支持版本管理
*   **连接池**: 优化的数据库连接池配置
*   **安全认证**: JWT token认证，bcrypt密码加密

## 📚 详细文档

*   **数据库设置**: 查看 [DATABASE_SETUP.md](DATABASE_SETUP.md) 获取详细的数据库配置指南
*   **项目结构**: 查看 [intro.md](intro.md) 了解项目架构

## ⚠️ 注意事项

*   **内存存储模式**: 服务重启后数据会丢失，仅适用于开发和测试
*   **JWT密钥**: 生产环境请务必使用强随机字符串
*   **数据库密码**: 生产环境请使用复杂密码并启用SSL连接


