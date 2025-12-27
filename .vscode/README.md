# VSCode 调试配置说明

本目录包含 Bison 项目的 VSCode 调试和任务配置。

## 📋 文件说明

- **launch.json** - 调试配置
- **tasks.json** - 任务配置
- **settings.json** - 工作区设置

## 🚀 可用的调试配置

### 单独调试

1. **API Server** - 调试 Go 后端 API 服务器
   - 端口: `8080`
   - 调试器: Go
   - 配置: 包含环境变量设置（OpenCost、Prometheus 等）

2. **Web UI** - 调试 React 前端应用
   - 端口: `5173` (Vite 默认)
   - 浏览器: Chrome
   - 自动启动: 是（preLaunchTask: dev-web）

3. **Documentation** - 调试 Docusaurus 文档站点
   - 端口: `3001` ⚡ （避免与 Web UI 冲突）
   - 浏览器: Chrome
   - 自动启动: 是（preLaunchTask: dev-docs）

### 组合调试

4. **Full Stack** - 同时调试 API Server + Web UI
   - 一键启动完整的开发环境
   - 包含 API 和前端
   - 文档站点保持独立调试

## 🔧 可用的任务

执行任务：`Cmd+Shift+P` → `Tasks: Run Task`

### 开发任务
- **dev-web** - 启动 Web UI 开发服务器
- **dev-docs** - 启动文档站点开发服务器

### 构建任务
- **build-api** - 构建 Go API 服务器
- **build-docs** - 构建文档站点

### 测试任务
- **test-api** - 运行 Go 测试

### 其他任务
- **install-deps** - 安装所有依赖
- **helm-lint** - 验证 Helm chart

## 🎯 使用方法

### 方法 1: 使用调试面板

1. 打开 VSCode 调试面板 (⇧⌘D)
2. 选择要运行的配置
3. 点击绿色播放按钮 ▶️

### 方法 2: 使用快捷键

1. 选择配置后，按 `F5` 开始调试
2. 按 `⇧F5` 停止调试

### 方法 3: 使用命令面板

1. `Cmd+Shift+P`
2. 输入 "Debug: Select and Start Debugging"
3. 选择配置

## 🌐 端口分配

| 服务 | 端口 | 说明 |
|------|------|------|
| API Server | 8080 | Go 后端服务 |
| Web UI | 5173 | Vite 开发服务器（自动） |
| Documentation | 3001 | Docusaurus 站点 |
| OpenCost | 30009 | (外部依赖) |
| Prometheus | 30090 | (外部依赖) |

## 💡 提示

### 快速启动文档调试

```bash
# 方法 1: 使用 Makefile
make dev-docs

# 方法 2: 使用 npm
cd website && npm run start

# 方法 3: 使用 VSCode 调试面板
# 选择 "Documentation" 配置并启动
```

### 调试 API Server

确保已设置正确的环境变量：
- `KUBECONFIG` - kubeconfig 文件路径
- `OPENCOST_URL` - OpenCost API 地址
- `PROMETHEUS_URL` - Prometheus 地址

### 同时调试多个服务

使用 "Full Stack" 配置可以同时启动：
- API Server (8080)
- Web UI (5173)

文档站点 (3001) 需要单独启动，使用 "Documentation" 配置

## 🔍 故障排查

### 端口已被占用

```bash
# 查看端口占用
lsof -i :3001
lsof -i :5173
lsof -i :8080

# 终止进程
kill -9 <PID>
```

### 任务启动失败

确保已安装所有依赖：
```bash
make install-deps
```

### 浏览器未自动打开

手动访问：
- Web UI: http://localhost:5173
- Documentation: http://localhost:3001
- API: http://localhost:8080

## 📚 相关文档

- [Makefile 命令](../Makefile) - 查看所有可用命令
- [项目文档](../website/docs/) - Docusaurus 文档源文件
- [Web UI](../web-ui/) - React 前端代码
- [API Server](../api-server/) - Go 后端代码
