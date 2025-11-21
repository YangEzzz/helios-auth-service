# `helios-auth-service-lite` 项目结构介绍


**核心特点：**
*   **精简高效：** 专注于核心功能，避免过度设计。
*   **职责分离：** 每个目录和文件都有明确的职责。
*   **登录/注册示例：** 包含完整的用户注册和登录流程（使用模拟数据存储）。
*   **无真实数据库连接：** 为了简化和快速演示，数据存储层使用了内存模拟，不依赖真实的PostgreSQL数据库。

---

## **1. 项目结构概览**

```
helios-auth-service-lite/
├── cmd/
│   └── api/
│       └── main.go                 # API服务的入口点
├── internal/
│   ├── auth/                       # 认证和授权逻辑
│   │   ├── service.go              # 认证服务接口和实现 (模拟)
│   │   └── handler.go              # 认证相关的HTTP处理器
│   ├── config/                     # 配置管理
│   │   └── config.go               # 应用配置结构和加载逻辑
│   ├── models/                     # 数据模型定义
│   │   └── user.go                 # 用户模型
│   ├── repository/                 # 数据访问层 (DAO/Repository)
│   │   └── user_repo.go            # 用户数据操作接口和实现 (内存模拟)
│   ├── services/                   # 业务逻辑层 (Service Layer)
│   │   └── user_service.go         # 用户业务逻辑 (模拟)
│   └── utils/                      # 通用工具函数
│       ├── jwt.go                  # JWT生成和验证
│       └── password.go             # 密码哈希和验证
├── go.mod                          # Go模块文件
├── go.sum                          # Go模块校验和文件
└── README.md                       # 项目说明 (待补充)
```

---

## **2. 目录与文件职责详解**

### **`cmd/` 目录**
*   **`cmd/api/main.go`**
    *   **职责：** 应用程序的入口点。负责初始化配置、模拟数据仓库、服务层、HTTP路由（使用Gin框架），并启动API服务器。
    *   **说明：** 这是整个后端服务的启动文件，协调各个组件协同工作。

### **`internal/` 目录**
*   **`internal/auth/`**
    *   **`service.go`**: 定义了 `AuthService` 接口及其模拟实现。负责用户登录时的密码验证和JWT Token的生成。它依赖 `repository` 层获取用户信息。
    *   **`handler.go`**: 定义了 `AuthHandler`，包含处理用户注册 (`/api/auth/register`) 和登录 (`/api/auth/login`) 请求的HTTP处理器。它依赖 `AuthService` 和 `UserService` 来完成业务逻辑。
*   **`internal/config/`**
    *   **`config.go`**: 负责加载应用程序的配置，如服务器端口和JWT密钥。为了简化，这里直接从环境变量或 `.env` 文件中读取。
*   **`internal/models/`**
    *   **`user.go`**: 定义了 `User` 结构体，代表系统中的用户实体。包含用户ID、邮箱、密码哈希等字段。
*   **`internal/repository/`**
    *   **`user_repo.go`**: 定义了 `UserRepository` 接口及其内存模拟实现 `mockUserRepository`。它模拟了数据库对用户数据的CRUD操作（创建、通过邮箱查询、通过ID查询），但不涉及真实的数据库连接。
*   **`internal/services/`**
    *   **`user_service.go`**: 定义了 `UserService` 接口及其模拟实现 `mockUserService`。负责用户注册的业务逻辑，如检查邮箱是否已注册、密码哈希等。它依赖 `UserRepository` 进行数据操作。
*   **`internal/utils/`**
    *   **`jwt.go`**: 提供了生成和解析JWT (JSON Web Token) 的工具函数，用于用户认证。
    *   **`password.go`**: 提供了密码哈希（使用bcrypt）和验证密码的工具函数，确保密码安全存储。

### **项目根目录**
*   **`go.mod`**: Go模块文件，定义了项目的模块路径和依赖关系。
*   **`go.sum`**: Go模块校验和文件，用于验证依赖的完整性。
*   **`README.md`**: 项目的说明文档，通常包含项目的介绍、如何运行、API文档等信息。

---

## **3. 如何使用**

1.  **解压文件：** 将 `helios-auth-service-lite.tar.gz` 解压到您希望的项目目录。
2.  **初始化Go模块：** 进入 `helios-auth-service-lite` 目录，运行 `go mod tidy` 下载所有依赖。
3.  **配置环境变量：** 在项目根目录创建 `.env` 文件，至少包含 `JWT_SECRET=your_super_secret_key` 和 `PORT=8080`。
4.  **运行服务：** 在项目根目录运行 `go run cmd/api/main.go`。
5.  **测试API：** 使用Postman或cURL测试 `/api/auth/register` 和 `/api/auth/login` 接口。

这份精简的项目结构和示例代码，旨在为您提供一个快速启动Go后端开发的起点，并清晰地展示了登录功能的实现逻辑。

