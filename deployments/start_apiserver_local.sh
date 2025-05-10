#!/bin/bash

# 获取脚本所在的目录，并推断项目根目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")" # 项目根目录 (im-go)

echo "正在启动 API 服务器..."
echo "项目根目录: $PROJECT_ROOT"

# 进入项目根目录
cd "$PROJECT_ROOT" || { echo "无法进入项目根目录 $PROJECT_ROOT"; exit 1; }

# 创建 bin 目录（如果不存在）
mkdir -p ./bin

# 编译 API 服务器
echo "正在编译 API 服务器 (cmd/apiserver/main.go)..."
go build -o ./bin/apiserver ./cmd/apiserver/main.go
if [ $? -ne 0 ]; then
    echo "API 服务器编译失败。"
    exit 1
fi
echo "API 服务器编译成功。输出: ./bin/apiserver"

# 运行 API 服务器
echo "正在运行 API 服务器 (./bin/apiserver)..."
./bin/apiserver 