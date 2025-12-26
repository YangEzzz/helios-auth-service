# 常用数据库方法参考

## 📚 GORM 常用方法速查表

### 1️⃣ 基础查询方法

#### 查询单条记录

```go
// 根据主键查询
var user models.User
db.First(&user, id)  // SELECT * FROM users WHERE id = ? LIMIT 1

// 根据条件查询第一条
db.Where("email = ?", "test@example.com").First(&user)

// 查询最后一条
db.Last(&user)

// 使用 Take（不排序）
db.Take(&user)
```

#### 查询多条记录

```go
// 查询所有记录
var users []models.User
db.Find(&users)

// 带条件查询
db.Where("status = ?", "active").Find(&users)

// 查询指定字段
db.Select("id", "email", "username").Find(&users)

// 排除字段
db.Omit("password_hash").Find(&users)
```

### 2️⃣ 条件查询

#### Where 条件

```go
// 字符串条件
db.Where("email = ?", "test@example.com").Find(&users)

// 多个条件
db.Where("status = ? AND role = ?", "active", "admin").Find(&users)

// Struct 条件（只查询非零值字段）
db.Where(&models.User{Status: "active", Role: "admin"}).Find(&users)

// Map 条件
db.Where(map[string]interface{}{"status": "active", "role": "admin"}).Find(&users)

// IN 查询
db.Where("id IN ?", []int{1, 2, 3}).Find(&users)

// LIKE 查询
db.Where("email LIKE ?", "%@example.com").Find(&users)

// BETWEEN
db.Where("created_at BETWEEN ? AND ?", startDate, endDate).Find(&users)
```

#### Or 条件

```go
db.Where("role = ?", "admin").Or("role = ?", "super_admin").Find(&users)
```

#### Not 条件

```go
db.Not("status = ?", "locked").Find(&users)
db.Not(map[string]interface{}{"status": "locked"}).Find(&users)
```

### 3️⃣ 排序、分页、限制

```go
// 排序
db.Order("created_at DESC").Find(&users)
db.Order("created_at DESC, id ASC").Find(&users)

// 分页
page := 1
pageSize := 10
offset := (page - 1) * pageSize
db.Offset(offset).Limit(pageSize).Find(&users)

// 限制数量
db.Limit(10).Find(&users)

// 去重
db.Distinct("email").Find(&users)
```

### 4️⃣ 聚合查询

```go
// 计数
var count int64
db.Model(&models.User{}).Count(&count)
db.Where("status = ?", "active").Count(&count)

// 分组计数
type Result struct {
    Status string
    Count  int64
}
var results []Result
db.Model(&models.User{}).Select("status, count(*) as count").Group("status").Scan(&results)

// 求和、平均值、最大值、最小值
db.Model(&models.Order{}).Select("sum(amount)").Row().Scan(&sum)
db.Model(&models.Order{}).Select("avg(amount)").Row().Scan(&avg)
db.Model(&models.Order{}).Select("max(amount)").Row().Scan(&max)
db.Model(&models.Order{}).Select("min(amount)").Row().Scan(&min)
```

### 5️⃣ 创建记录

```go
// 创建单条
user := models.User{Email: "test@example.com", Username: "test"}
db.Create(&user)  // user.ID 会被自动填充

// 批量创建
users := []models.User{
    {Email: "user1@example.com"},
    {Email: "user2@example.com"},
}
db.Create(&users)

// 创建并选择字段
db.Select("Email", "Username").Create(&user)

// 创建并忽略字段
db.Omit("CreatedAt").Create(&user)
```

### 6️⃣ 更新记录

```go
// 更新单个字段
db.Model(&user).Update("status", "active")


// 更新选定字段
db.Model(&user).Select("status", "role").Updates(map[string]interface{}{
    "status": "active",
    "role":   "admin",
})
```

### 7️⃣ 删除记录

```go
// 删除单条（需要主键）
db.Delete(&user)
db.Delete(&user, id)

// 根据条件删除
db.Where("status = ?", "inactive").Delete(&models.User{})

// 批量删除
db.Delete(&models.User{}, []int{1, 2, 3})

// 软删除（需要在模型中添加 DeletedAt 字段）
type User struct {
    ID        uint
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
db.Delete(&user)  // 实际执行 UPDATE users SET deleted_at = NOW() WHERE id = ?

// 永久删除（跳过软删除）
db.Unscoped().Delete(&user)

// 查询包括软删除的记录
db.Unscoped().Find(&users)
```

### 8️⃣ 关联查询

```go
// Preload 预加载
type User struct {
    ID       uint
    Projects []Project
}
db.Preload("Projects").Find(&users)

// 嵌套预加载
db.Preload("Projects.Tasks").Find(&users)

// 条件预加载
db.Preload("Projects", "status = ?", "active").Find(&users)

// Joins 连接查询
db.Joins("LEFT JOIN projects ON projects.user_id = users.id").Find(&users)

// 自定义 Join 条件
db.Joins("LEFT JOIN projects ON projects.user_id = users.id AND projects.status = ?", "active").Find(&users)
```

### 9️⃣ 原生 SQL

```go
// 原生查询
type Result struct {
    Email string
    Count int
}
var results []Result
db.Raw("SELECT email, count(*) as count FROM users GROUP BY email").Scan(&results)

// 原生执行
db.Exec("UPDATE users SET status = ? WHERE id = ?", "active", 1)

// 使用命名参数
db.Raw("SELECT * FROM users WHERE email = @email",
    sql.Named("email", "test@example.com")).Scan(&user)
```

### 🔟 事务处理

```go
// 自动事务
err := db.Transaction(func(tx *gorm.DB) error {
    // 在事务中执行操作
    if err := tx.Create(&user).Error; err != nil {
        return err  // 返回错误会自动回滚
    }

    if err := tx.Create(&project).Error; err != nil {
        return err
    }

    return nil  // 返回 nil 会自动提交
})

// 手动事务
tx := db.Begin()

if err := tx.Create(&user).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Create(&project).Error; err != nil {
    tx.Rollback()
    return err
}

tx.Commit()
```

### 1️⃣1️⃣ 高级查询

```go
// 子查询
db.Where("amount > (?)", db.Table("orders").Select("AVG(amount)")).Find(&orders)

// Scopes（可复用查询逻辑）
func ActiveUsers(db *gorm.DB) *gorm.DB {
    return db.Where("status = ?", "active")
}

func AdminUsers(db *gorm.DB) *gorm.DB {
    return db.Where("role = ?", "admin")
}

db.Scopes(ActiveUsers, AdminUsers).Find(&users)

// FirstOrCreate（查找或创建）
db.Where(models.User{Email: "test@example.com"}).FirstOrCreate(&user)

// FirstOrInit（查找或初始化，不保存）
db.Where(models.User{Email: "test@example.com"}).FirstOrInit(&user)

// Pluck（查询单列）
var emails []string
db.Model(&models.User{}).Pluck("email", &emails)

// Scan（扫描到自定义结构）
type UserInfo struct {
    Email    string
    Username string
}
var userInfos []UserInfo
db.Model(&models.User{}).Select("email", "username").Scan(&userInfos)
```

### 1️⃣2️⃣ 钩子方法

```go
// BeforeCreate
func (u *User) BeforeCreate(tx *gorm.DB) error {
    u.ID = uuid.New()
    return nil
}

// AfterCreate
func (u *User) AfterCreate(tx *gorm.DB) error {
    // 发送欢迎邮件等
    return nil
}

// BeforeUpdate
func (u *User) BeforeUpdate(tx *gorm.DB) error {
    u.UpdatedAt = time.Now()
    return nil
}

// BeforeDelete
func (u *User) BeforeDelete(tx *gorm.DB) error {
    // 清理关联数据
    return nil
}
```

### 1️⃣3️⃣ 性能优化

```go
// 批量插入（使用 CreateInBatches）
db.CreateInBatches(users, 100)  // 每批 100 条

// 使用索引提示
db.Clauses(hints.UseIndex("idx_user_email")).Find(&users)

// 只查询需要的字段
db.Select("id", "email").Find(&users)

// 使用 Find 代替循环查询
// ❌ 不好的做法
for _, id := range ids {
    db.First(&user, id)
}

// ✅ 好的做法
db.Find(&users, ids)

// 预编译语句
stmt := db.Session(&gorm.Session{PrepareStmt: true})
for i := 0; i < 100; i++ {
    stmt.Where("id = ?", i).First(&user)
}
```

### 1️⃣4️⃣ 错误处理

```go
// 检查记录是否存在
if err := db.First(&user, id).Error; err != nil {
    if errors.Is(err, gorm.ErrRecordNotFound) {
        // 记录不存在
    } else {
        // 其他错误
    }
}

// 检查是否影响了行
result := db.Where("status = ?", "inactive").Delete(&models.User{})
if result.RowsAffected == 0 {
    // 没有记录被删除
}
```

## 🎯 实用示例

### 分页查询完整示例

```go
type PaginationResult struct {
    Data       interface{} `json:"data"`
    Total      int64       `json:"total"`
    Page       int         `json:"page"`
    PageSize   int         `json:"page_size"`
    TotalPages int         `json:"total_pages"`
}

func GetUsersPaginated(db *gorm.DB, page, pageSize int) (*PaginationResult, error) {
    var users []models.User
    var total int64

    // 计算总数
    if err := db.Model(&models.User{}).Count(&total).Error; err != nil {
        return nil, err
    }

    // 查询数据
    offset := (page - 1) * pageSize
    if err := db.Offset(offset).Limit(pageSize).Find(&users).Error; err != nil {
        return nil, err
    }

    totalPages := int(total) / pageSize
    if int(total)%pageSize != 0 {
        totalPages++
    }

    return &PaginationResult{
        Data:       users,
        Total:      total,
        Page:       page,
        PageSize:   pageSize,
        TotalPages: totalPages,
    }, nil
}
```

### 搜索和过滤示例

```go
func SearchUsers(db *gorm.DB, keyword string, status string, role string) ([]models.User, error) {
    var users []models.User
    query := db.Model(&models.User{})

    // 关键词搜索
    if keyword != "" {
        query = query.Where("email LIKE ? OR username LIKE ?",
            "%"+keyword+"%", "%"+keyword+"%")
    }

    // 状态过滤
    if status != "" {
        query = query.Where("status = ?", status)
    }

    // 角色过滤
    if role != "" {
        query = query.Where("role = ?", role)
    }

    if err := query.Find(&users).Error; err != nil {
        return nil, err
    }

    return users, nil
}
```

### 批量操作示例

```go
// 批量更新状态
func BatchUpdateUserStatus(db *gorm.DB, userIDs []string, status string) error {
    return db.Model(&models.User{}).
        Where("id IN ?", userIDs).
        Update("status", status).Error
}

// 批量删除
func BatchDeleteUsers(db *gorm.DB, userIDs []string) error {
    return db.Delete(&models.User{}, userIDs).Error
}
```

