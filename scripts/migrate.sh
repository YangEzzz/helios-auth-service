#!/bin/bash

# 数据库迁移脚本
# 使用方法:
# ./scripts/migrate.sh up      # 执行所有向上迁移
# ./scripts/migrate.sh down    # 执行所有向下迁移
# ./scripts/migrate.sh status  # 查看当前状态

set -e

# 检查参数
if [ $# -eq 0 ]; then
    echo "Usage: $0 {up|down|status|force VERSION}"
    echo "Examples:"
    echo "  $0 up           # Run all up migrations"
    echo "  $0 down         # Run all down migrations"
    echo "  $0 status       # Show current migration status"
    echo "  $0 force 1      # Force set version to 1"
    exit 1
fi

COMMAND=$1

# 切换到项目根目录
cd "$(dirname "$0")/.."

case $COMMAND in
    "up")
        echo "Running up migrations..."
        go run scripts/migrate.go -direction=up
        ;;
    "down")
        echo "Running down migrations..."
        go run scripts/migrate.go -direction=down
        ;;
    "status")
        echo "Checking migration status..."
        go run scripts/migrate.go -direction=up -steps=0
        ;;
    "force")
        if [ $# -ne 2 ]; then
            echo "Usage: $0 force VERSION"
            exit 1
        fi
        VERSION=$2
        echo "Forcing migration to version $VERSION..."
        go run scripts/migrate.go -force=$VERSION
        ;;
    *)
        echo "Invalid command: $COMMAND"
        echo "Use: up, down, status, or force VERSION"
        exit 1
        ;;
esac

echo "Migration operation completed."
