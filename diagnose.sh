#!/bin/bash
# Easy Proxies 配置诊断脚本
# 用于检查 Docker 环境中的配置文件权限问题

set -e

echo "=========================================="
echo "Easy Proxies 配置诊断工具"
echo "=========================================="
echo

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

success() {
    echo -e "${GREEN}✓${NC} $1"
}

warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

error() {
    echo -e "${RED}✗${NC} $1"
}

# 1. 检查数据目录
echo "1. 检查数据目录..."
DATA_DIR="./data"

if [ ! -d "$DATA_DIR" ]; then
    error "数据目录 $DATA_DIR 不存在"
    echo "   解决方法: mkdir -p $DATA_DIR"
    exit 1
else
    success "数据目录存在: $DATA_DIR"
fi

# 2. 检查目录权限
echo
echo "2. 检查目录权限..."
DIR_PERMS=$(stat -f "%Sp" "$DATA_DIR" 2>/dev/null || stat -c "%A" "$DATA_DIR" 2>/dev/null)
DIR_OWNER=$(stat -f "%Su:%Sg" "$DATA_DIR" 2>/dev/null || stat -c "%U:%G" "$DATA_DIR" 2>/dev/null)

echo "   目录权限: $DIR_PERMS"
echo "   目录所有者: $DIR_OWNER"

if [ -w "$DATA_DIR" ]; then
    success "目录可写"
else
    error "目录不可写"
    echo "   解决方法: chmod 755 $DATA_DIR"
fi

# 3. 检查配置文件
echo
echo "3. 检查配置文件..."

CONFIG_FILE="$DATA_DIR/config.yaml"
NODES_FILE="$DATA_DIR/nodes.txt"

check_file() {
    local file=$1
    local name=$2

    if [ ! -f "$file" ]; then
        warning "$name 不存在: $file"
        echo "   首次启动时会自动创建"
        return
    fi

    FILE_PERMS=$(stat -f "%Sp" "$file" 2>/dev/null || stat -c "%A" "$file" 2>/dev/null)
    FILE_OWNER=$(stat -f "%Su:%Sg" "$file" 2>/dev/null || stat -c "%U:%G" "$file" 2>/dev/null)

    echo "   $name:"
    echo "     路径: $file"
    echo "     权限: $FILE_PERMS"
    echo "     所有者: $FILE_OWNER"

    if [ -w "$file" ]; then
        success "   $name 可写"
    else
        error "   $name 不可写"
        echo "     解决方法: chmod 644 $file"
    fi
}

check_file "$CONFIG_FILE" "配置文件"
check_file "$NODES_FILE" "节点文件"

# 4. 检查 Docker 配置
echo
echo "4. 检查 Docker 配置..."

if [ -f "docker-compose.yml" ]; then
    success "找到 docker-compose.yml"

    # 检查卷映射
    if grep -q "./data:/etc/easy_proxies" docker-compose.yml; then
        success "数据目录已正确映射"
    else
        warning "未找到标准的数据目录映射"
        echo "   期望配置: ./data:/etc/easy_proxies"
    fi

    # 检查用户配置
    if grep -q "user:" docker-compose.yml; then
        USER_CONFIG=$(grep "user:" docker-compose.yml | head -1)
        echo "   用户配置: $USER_CONFIG"

        if echo "$USER_CONFIG" | grep -q '${UID:-10001}:${GID:-10001}'; then
            success "使用动态 UID/GID 配置"
            echo "   当前用户 UID:GID = $(id -u):$(id -g)"
        fi
    fi
else
    warning "未找到 docker-compose.yml"
fi

# 5. 给出建议
echo
echo "=========================================="
echo "修复建议"
echo "=========================================="
echo

if [ ! -w "$DATA_DIR" ] || ([ -f "$CONFIG_FILE" ] && [ ! -w "$CONFIG_FILE" ]) || ([ -f "$NODES_FILE" ] && [ ! -w "$NODES_FILE" ]); then
    echo "发现权限问题，请执行以下命令修复："
    echo
    echo "  # 修复目录权限"
    echo "  chmod 755 $DATA_DIR"
    echo
    echo "  # 修复文件权限（如果文件已存在）"
    echo "  [ -f $CONFIG_FILE ] && chmod 644 $CONFIG_FILE"
    echo "  [ -f $NODES_FILE ] && chmod 644 $NODES_FILE"
    echo
    echo "  # 修复所有者（使用当前用户）"
    echo "  chown -R $(id -u):$(id -g) $DATA_DIR"
    echo
fi

echo "Docker 启动建议："
echo
echo "  # 方式 1: 使用 docker-compose（推荐）"
echo "  UID=$(id -u) GID=$(id -g) docker-compose up -d"
echo
echo "  # 方式 2: 直接使用 docker run"
echo "  docker run -d \\"
echo "    --name easy_proxies \\"
echo "    --user \$(id -u):\$(id -g) \\"
echo "    --network host \\"
echo "    -v ./data:/etc/easy_proxies \\"
echo "    -v ./logs:/app/logs \\"
echo "    ghcr.io/jasonwong1991/easy_proxies:latest"
echo

echo "=========================================="
echo "诊断完成"
echo "=========================================="
