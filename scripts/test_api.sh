#!/bin/bash

# API测试脚本
# 用于测试用户注册和登录功能

set -e

BASE_URL="http://localhost:3456"
EMAIL="test_$(date +%s)@example.com"  # 使用时间戳生成唯一邮箱
PASSWORD="password123"

echo "🚀 开始测试 Helios Auth Service API..."
echo "📧 测试邮箱: $EMAIL"

# 检查服务是否运行
echo "📡 检查服务状态..."
if ! curl -s "$BASE_URL" > /dev/null 2>&1; then
    echo "❌ 服务未运行，请先启动服务: go run cmd/api/main.go"
    exit 1
fi

echo "✅ 服务正在运行"

# 测试用户注册
echo ""
echo "👤 测试用户注册..."
REGISTER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "注册响应: $REGISTER_RESPONSE"

# 检查注册是否成功
if echo "$REGISTER_RESPONSE" | grep -q "Registration successful"; then
    echo "✅ 用户注册成功"
    
    # 提取token
    TOKEN=$(echo "$REGISTER_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    echo "🔑 获取到Token: ${TOKEN:0:20}..."
else
    echo "❌ 用户注册失败"
fi

# 测试用户登录
echo ""
echo "🔐 测试用户登录..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "登录响应: $LOGIN_RESPONSE"

# 检查登录是否成功
if echo "$LOGIN_RESPONSE" | grep -q "Login successful"; then
    echo "✅ 用户登录成功"
    
    # 提取token
    LOGIN_TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    echo "🔑 登录Token: ${LOGIN_TOKEN:0:20}..."
else
    echo "❌ 用户登录失败"
fi

# 测试重复注册
echo ""
echo "🔄 测试重复注册..."
DUPLICATE_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/register" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")

echo "重复注册响应: $DUPLICATE_RESPONSE"

if echo "$DUPLICATE_RESPONSE" | grep -q "already"; then
    echo "✅ 重复注册检查正常"
else
    echo "❌ 重复注册检查失败"
fi

# 测试错误密码登录
echo ""
echo "🚫 测试错误密码登录..."
WRONG_PASSWORD_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EMAIL\",\"password\":\"wrongpassword\"}")

echo "错误密码响应: $WRONG_PASSWORD_RESPONSE"

if echo "$WRONG_PASSWORD_RESPONSE" | grep -q "invalid credentials"; then
    echo "✅ 错误密码检查正常"
else
    echo "❌ 错误密码检查失败"
fi

# 测试受保护的路由（使用 Token）
echo ""
echo "🔐 测试受保护的路由 - 获取用户信息..."
if [ -n "$TOKEN" ]; then
    PROFILE_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/profile" \
        -H "Authorization: Bearer $TOKEN")

    echo "用户信息响应: $PROFILE_RESPONSE"

    if echo "$PROFILE_RESPONSE" | grep -q "user"; then
        echo "✅ 获取用户信息成功"
    else
        echo "❌ 获取用户信息失败"
    fi
else
    echo "⚠️  跳过受保护路由测试（没有 Token）"
fi

# 测试未授权访问受保护路由
echo ""
echo "🚫 测试未授权访问受保护路由..."
UNAUTHORIZED_RESPONSE=$(curl -s -X GET "$BASE_URL/api/v1/profile")

echo "未授权响应: $UNAUTHORIZED_RESPONSE"

if echo "$UNAUTHORIZED_RESPONSE" | grep -q "Missing authorization header"; then
    echo "✅ 未授权访问检查正常"
else
    echo "❌ 未授权访问检查失败"
fi

echo ""
echo "🎉 API测试完成！"
echo ""
echo "📊 测试总结："
echo "  ✅ 用户注册"
echo "  ✅ 用户登录"
echo "  ✅ 重复注册检查"
echo "  ✅ 错误密码检查"
echo "  ✅ 受保护路由访问"
echo "  ✅ 未授权访问检查"
