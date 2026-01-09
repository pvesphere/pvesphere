#!/bin/bash

# PVESphere Docker 构建脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
DEFAULT_FRONTEND_DIR="/Users/ztwork/server/git-repos/pvesphere-ui"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BUILD_DIR="$SCRIPT_DIR/build"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}PVESphere Docker 构建${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 帮助信息
show_help() {
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -f, --frontend DIR   指定前端代码目录（默认: $DEFAULT_FRONTEND_DIR）"
    echo "  -l, --link           创建符号链接而不是复制（节省空间）"
    echo "  -c, --clean          清理本地前端缓存"
    echo "  -b, --backend        只构建后端服务（api + controller）"
    echo "  -a, --all            构建所有服务（前端 + 后端）"
    echo "  -h, --help           显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                          # 使用默认目录构建前端"
    echo "  $0 -f /path/to/pvesphere-ui # 指定前端目录"
    echo "  $0 -b                       # 只构建后端服务"
    echo "  $0 -a                       # 构建所有服务"
    echo "  $0 -l                       # 使用符号链接"
    echo "  $0 -c                       # 清理缓存"
}

# 清理函数
cleanup() {
    echo -e "${YELLOW}清理本地前端缓存...${NC}"
    cd "$BUILD_DIR"
    rm -rf .frontend-local
    echo -e "${GREEN}✓ 清理完成${NC}"
}

# 解析参数
USE_LINK=false
FRONTEND_DIR="$DEFAULT_FRONTEND_DIR"

while [[ $# -gt 0 ]]; do
    case $1 in
        -f|--frontend)
            FRONTEND_DIR="$2"
            shift 2
            ;;
        -l|--link)
            USE_LINK=true
            shift
            ;;
        -c|--clean)
            cleanup
            exit 0
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知选项 $1${NC}"
            show_help
            exit 1
            ;;
    esac
done

cd "$BUILD_DIR"

# 检查前端目录
if [ ! -d "$FRONTEND_DIR" ]; then
    echo -e "${RED}✗ 前端目录不存在: $FRONTEND_DIR${NC}"
    echo ""
    echo "请指定正确的前端目录:"
    echo "  $0 -f /path/to/pvesphere-ui"
    exit 1
fi

echo -e "${GREEN}✓ 找到前端目录: $FRONTEND_DIR${NC}"

# 检查 package.json
if [ ! -f "$FRONTEND_DIR/package.json" ]; then
    echo -e "${RED}✗ 前端目录中找不到 package.json${NC}"
    echo "请确保这是正确的前端项目目录"
    exit 1
fi

# 准备本地前端代码
echo -e "${YELLOW}准备本地前端代码...${NC}"
rm -rf .frontend-local

if [ "$USE_LINK" = true ]; then
    echo -e "${BLUE}创建符号链接...${NC}"
    ln -s "$FRONTEND_DIR" .frontend-local
else
    echo -e "${BLUE}复制前端代码...${NC}"
    mkdir -p .frontend-local
    # 排除 node_modules、dist 等构建产物和版本控制目录
    # 注意：保留 .npmrc 等配置文件
    rsync -av --exclude='node_modules' \
              --exclude='dist' \
              --exclude='.git' \
              --exclude='.idea' \
              --exclude='.vscode' \
              --exclude='.cursor' \
              --exclude='*.log' \
              --include='.npmrc' \
              --include='.env*' \
              "$FRONTEND_DIR/" .frontend-local/
fi

# 验证复制结果
if [ ! -f ".frontend-local/package.json" ]; then
    echo -e "${RED}✗ 前端代码准备失败：找不到 package.json${NC}"
    exit 1
fi

file_count=$(find .frontend-local -type f | wc -l)
echo -e "${GREEN}✓ 前端代码准备完成（$file_count 个文件）${NC}"
echo ""

# 构建 Docker 镜像
echo -e "${YELLOW}开始构建 Docker 镜像...${NC}"
cd "$PROJECT_ROOT"

docker-compose -f deploy/docker-compose/docker-compose.yml build frontend

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✓ 构建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "启动服务:"
echo "  cd deploy/docker-compose"
echo "  docker-compose up -d"
echo ""
echo "清理本地缓存:"
echo "  $0 -c"
