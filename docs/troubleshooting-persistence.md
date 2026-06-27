# 配置持久化故障排查指南

## 问题描述

在 Docker 环境中修改配置后，重启或重建容器时配置被重置。

## 根本原因

配置持久化失败通常由以下原因引起：

1. **文件权限问题**：容器内的用户没有写入宿主机目录的权限
2. **卷映射错误**：配置文件没有正确映射到宿主机
3. **文件系统只读**：某些环境下文件系统被挂载为只读
4. **挂载不存在的文件 → Docker 创建目录**：见下文专项小节

## 快速诊断

```bash
# 检查 data 目录结构和权限
ls -la data/
[ -f data/config.yaml ]  || echo "缺少 data/config.yaml"
[ -d data/config.yaml ]   && echo "异常：data/config.yaml 是目录（见下文 'nodes.txt/config.yaml 变成了目录'）"
[ -d data/nodes.txt ]    && echo "异常：data/nodes.txt 是目录（见下文 'nodes.txt/config.yaml 变成了目录'）"
```

该检查会判断：
- 数据目录是否存在和可写
- 配置文件权限
- 是否踩了"挂载不存在的文件 → Docker 创建目录"的坑
- 给出修复建议

## 解决方案

### 方案 1：修复文件权限（推荐）

```bash
# 创建数据目录
mkdir -p data

# 设置正确的所有者和权限
chown -R $(id -u):$(id -g) data
chmod 755 data

# 如果配置文件已存在，修复其权限
[ -f data/config.yaml ] && chmod 644 data/config.yaml
[ -f data/nodes.txt ] && chmod 644 data/nodes.txt
```

### 方案 2：使用正确的 UID/GID 启动容器

**使用 docker-compose（推荐）：**

```bash
# 传递当前用户的 UID 和 GID
UID=$(id -u) GID=$(id -g) docker-compose up -d
```

或者在 `docker-compose.yml` 中设置：

```yaml
services:
  easy_proxies:
    user: "${UID:-10001}:${GID:-10001}"
    volumes:
      - ./data:/etc/easy_proxies
      - ./logs:/app/logs
```

**使用 docker run：**

```bash
docker run -d \
  --name easy_proxies \
  --user $(id -u):$(id -g) \
  --network host \
  -v ./data:/etc/easy_proxies \
  -v ./logs:/app/logs \
  ghcr.io/jasonwong1991/easy_proxies:latest
```

### 方案 3：验证卷映射

确保 `docker-compose.yml` 或 `docker run` 命令中包含正确的卷映射：

```yaml
volumes:
  - ./data:/etc/easy_proxies  # 配置目录
  - ./logs:/app/logs          # 日志目录（可选）
```

**重要**：使用相对路径 `./data` 而不是绝对路径，这样配置会保存在项目目录下。

## 验证配置持久化

1. **通过 WebUI 修改配置**
   - 访问 `http://localhost:9091`
   - 修改设置或添加节点
   - 检查 `./data/config.yaml` 是否更新

2. **检查文件修改时间**

```bash
# 查看配置文件最后修改时间
ls -lh data/config.yaml data/nodes.txt
```

3. **查看日志**

容器日志中会显示保存结果：

```bash
docker-compose logs -f easy_proxies
```

成功保存时会看到：
```
✅ Saved 2 nodes to /etc/easy_proxies/nodes.txt
✅ Saved 1 inline nodes to /etc/easy_proxies/config.yaml
```

失败时会看到：
```
ERROR: config file not writable: ... (check file permissions and Docker volume mounts)
```

## 常见错误信息及解决

### 错误 1: "config file not writable: permission denied"

**原因**：容器用户没有写入权限

**解决**：
```bash
chown -R $(id -u):$(id -g) data
chmod 755 data
```

### 错误 2: "config file not writable: file is read-only"

**原因**：文件被标记为只读

**解决**：
```bash
chmod 644 data/config.yaml data/nodes.txt
```

### 错误 3: "directory not writable"

**原因**：数据目录不可写

**解决**：
```bash
chmod 755 data
```

### 错误 4: 配置保存成功但重启后丢失

**原因**：卷映射错误，配置保存在容器内而不是宿主机

**解决**：
1. 停止容器：`docker-compose down`
2. 检查 `docker-compose.yml` 中的 volumes 配置
3. 确保使用 `./data:/etc/easy_proxies` 映射
4. 重新启动：`UID=$(id -u) GID=$(id -g) docker-compose up -d`

### 错误 5: "nodes.txt is a directory, not a file" / "config.yaml is a directory, not a file"

**原因**：使用了文件级挂载（`-v ./data/nodes.txt:/etc/easy_proxies/nodes.txt`），但宿主机上 `./data/nodes.txt` 这个文件启动前不存在。Docker 此时不会报错，而是在宿主机上创建一个名为 `nodes.txt` 的**目录**并挂载进容器，于是容器内 `/etc/easy_proxies/nodes.txt` 变成了一个目录，应用读取/写入时失败。`config.yaml` 同理。

**判别**：
```bash
ls -la data/
# 若看到这样的行，即踩坑：
# drwxr-xr-x  nodes.txt
# drwxr-xr-x  config.yaml
```

**解决**（任选其一）：

- 方式 A（推荐，目录挂载 + 首启动自动生成文件）：
  ```bash
  docker compose down
  rm -rf data/nodes.txt data/config.yaml   # 删掉被错误创建成目录的路径
  docker compose up -d                      # 由 entrypoint 自动生成真正的文件
  ```

- 方式 B（保留文件级挂载，预先在宿主机创建文件）：
  ```bash
  docker compose down
  rm -rf data/nodes.txt data/config.yaml
  cp config.example.yaml data/config.yaml
  touch data/nodes.txt
  chown -R $(id -u):$(id -g) data
  docker compose up -d
  ```

> 注：`entrypoint.sh` 现在会检测到这个情况并立即退出、给出上述修复指引，而不是让应用以晦涩的运行时错误崩溃。

## Docker 最佳实践

### 推荐的目录结构

```
easy_proxies/
├── docker-compose.yml
├── data/                    # 配置目录（持久化）
│   ├── config.yaml         # 主配置文件
│   └── nodes.txt           # 节点文件
└── logs/                    # 日志目录（可选）
    └── easy_proxies.log
```

### 推荐的 docker-compose.yml

```yaml
services:
  easy_proxies:
    image: ghcr.io/jasonwong1991/easy_proxies:latest
    container_name: easy_proxies
    restart: unless-stopped
    network_mode: host
    user: "${UID:-10001}:${GID:-10001}"
    volumes:
      - ./data:/etc/easy_proxies
      - ./logs:/app/logs
```

### 启动命令

```bash
# 创建必要的目录
mkdir -p data logs

# 设置权限
chown -R $(id -u):$(id -g) data logs

# 启动容器
UID=$(id -u) GID=$(id -g) docker-compose up -d

# 查看日志
docker-compose logs -f
```

## 配置文件说明

### config.yaml

存储应用的核心配置，包括：
- 监听端口和地址
- 代理池设置
- 订阅配置
- inline 节点（直接在 config.yaml 中定义的节点）

### nodes.txt

存储从订阅获取的节点和通过 WebUI 添加的节点（当存在订阅时）。格式为每行一个代理 URI。

**重要**：
- 如果同时配置了 `nodes`（inline）和 `subscriptions`，两者会合并使用
- inline 节点不会被订阅更新覆盖
- 订阅节点保存在 `nodes.txt` 中

## 高级故障排查

### 检查容器内的权限

```bash
# 进入容器
docker-compose exec easy_proxies sh

# 检查配置目录
ls -la /etc/easy_proxies/

# 检查当前用户
id

# 尝试写入测试
touch /etc/easy_proxies/test.txt
rm /etc/easy_proxies/test.txt
```

### 查看详细日志

```bash
# 查看完整日志
docker-compose logs --tail=100 easy_proxies

# 实时跟踪日志
docker-compose logs -f easy_proxies
```

### 手动验证配置保存

```bash
# 1. 修改配置前记录哈希
md5sum data/config.yaml

# 2. 通过 WebUI 修改配置

# 3. 修改后再次检查哈希
md5sum data/config.yaml

# 如果哈希值改变，说明配置成功保存
```

## 需要帮助？

如果以上方法都无法解决问题，请：

1. 运行诊断命令并保存输出（见上"快速诊断"小节）：
   ```bash
   ls -la data/ > diagnose.log
   docker compose logs easy_proxies > container.log
   ```
2. 收集容器日志：`docker-compose logs > container.log`
3. 提供以下信息：
   - 操作系统和 Docker 版本
   - `docker-compose.yml` 内容
   - 诊断脚本输出
   - 容器日志
4. 在 GitHub Issues 中提交问题
