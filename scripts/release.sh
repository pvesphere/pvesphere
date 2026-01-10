#!/bin/bash

# PveSphere v1.0.0-rc01 Release Script
# 此脚本帮助自动化发布流程

set -e

VERSION="1.0.0-rc01"
BACKEND_DIR="/Users/ztwork/server/git-repos/pvesphere"
FRONTEND_DIR="/Users/ztwork/server/git-repos/pvesphere-ui"
DOCS_DIR="/Users/ztwork/server/git-repos/pvesphere-docs"

echo "========================================="
echo "  PveSphere v${VERSION} Release Script"
echo "   (Release Candidate 1)"
echo "========================================="
echo ""

# 颜色输出
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# 检查仓库是否存在
check_repo() {
    local dir=$1
    local name=$2
    if [ ! -d "$dir" ]; then
        echo -e "${RED}✗ 错误: $name 仓库不存在: $dir${NC}"
        exit 1
    fi
    echo -e "${GREEN}✓ $name 仓库存在${NC}"
}

echo "1️⃣  检查仓库..."
check_repo "$BACKEND_DIR" "后端"
check_repo "$FRONTEND_DIR" "前端"
check_repo "$DOCS_DIR" "文档"
echo ""

# 检查 git 状态
check_git_status() {
    local dir=$1
    local name=$2
    cd "$dir"
    if [[ -n $(git status -s) ]]; then
        echo -e "${YELLOW}⚠ 警告: $name 仓库有未提交的更改${NC}"
        git status -s
        read -p "是否继续? (y/n) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        echo -e "${GREEN}✓ $name 仓库状态干净${NC}"
    fi
}

echo "2️⃣  检查 Git 状态..."
check_git_status "$BACKEND_DIR" "后端"
check_git_status "$FRONTEND_DIR" "前端"
check_git_status "$DOCS_DIR" "文档"
echo ""

# 创建标签
create_tag() {
    local dir=$1
    local name=$2
    cd "$dir"
    
    echo -e "${YELLOW}为 $name 仓库创建标签 v${VERSION}...${NC}"
    
    # 检查标签是否已存在
    if git rev-parse "v${VERSION}" >/dev/null 2>&1; then
        echo -e "${RED}✗ 标签 v${VERSION} 已存在${NC}"
        read -p "是否删除并重新创建? (y/n) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            git tag -d "v${VERSION}"
            git push origin ":refs/tags/v${VERSION}" 2>/dev/null || true
        else
            return
        fi
    fi
    
    git tag -a "v${VERSION}" -m "Release v${VERSION}"
    echo -e "${GREEN}✓ 标签创建成功${NC}"
    
    read -p "是否推送标签到远程? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        git push origin "v${VERSION}"
        echo -e "${GREEN}✓ 标签已推送到远程${NC}"
    fi
}

echo "3️⃣  创建 Git 标签..."
read -p "是否为所有仓库创建并推送标签 v${VERSION}? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    create_tag "$BACKEND_DIR" "后端"
    create_tag "$FRONTEND_DIR" "前端"
    create_tag "$DOCS_DIR" "文档"
fi
echo ""

# Docker 镜像构建
echo "4️⃣  构建 Docker 镜像..."
read -p "是否构建 Docker 镜像? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    cd "$BACKEND_DIR"
    echo -e "${YELLOW}构建 Docker 镜像...${NC}"
    make docker-build
    echo -e "${GREEN}✓ Docker 镜像构建完成${NC}"
fi
echo ""

# 推送 Docker 镜像
echo "5️⃣  推送 Docker 镜像..."
read -p "是否推送 Docker 镜像到仓库? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    read -p "请输入镜像仓库地址 (例如: docker.io/pvesphere): " REGISTRY
    if [ -n "$REGISTRY" ]; then
        cd "$BACKEND_DIR"
        echo -e "${YELLOW}推送镜像到 $REGISTRY...${NC}"
        make docker-push REGISTRY="$REGISTRY"
        echo -e "${GREEN}✓ Docker 镜像推送完成${NC}"
    else
        echo -e "${YELLOW}⚠ 跳过镜像推送${NC}"
    fi
fi
echo ""

echo "========================================="
echo -e "${GREEN}✓ 发布流程完成！${NC}"
echo "========================================="
