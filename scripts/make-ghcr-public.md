# 将 GHCR Package 设为公开

由于你无法通过 OCI 路径拉取 Helm chart (`Error: invalid_reference: invalid repository` 或 `403 Forbidden`)，这是因为 GitHub Container Registry 的包默认是私有的。

## 手动设置 Package 为 Public

1. **访问 GitHub Packages**:
   - 进入 https://github.com/SuperMarioYL?tab=packages
   - 或者直接访问仓库首页，点击右侧的 "Packages"

2. **找到 charts/bison package**:
   - 如果存在，点击进入 `charts/bison` package

3. **修改可见性**:
   - 点击 **Package settings** (右上角齿轮图标)
   - 滚动到底部找到 **Danger Zone**
   - 点击 **Change visibility**
   - 选择 **Public**
   - 确认更改

## 如果找不到 charts/bison Package

说明 Helm chart 推送到 GHCR 失败了。检查步骤:

1. **检查 GitHub Actions 运行日志**:
   ```
   https://github.com/SuperMarioYL/Bison/actions
   ```
   - 找到最近的 "Release" workflow run (v0.0.7)
   - 查看 "Publish to Helm Repository (GHCR)" job 的日志
   - 检查是否有错误信息

2. **常见失败原因**:
   - 权限不足 (GITHUB_TOKEN 没有 `packages: write` 权限)
   - OCI 路径错误
   - Helm 登录失败

## 临时解决方案

在 GHCR OCI 路径修复之前，使用 GitHub Releases:

```bash
# 列出所有可用版本
curl -s https://api.github.com/repos/SuperMarioYL/Bison/releases | grep tag_name

# 下载特定版本
VERSION=0.0.7
wget https://github.com/SuperMarioYL/Bison/releases/download/v${VERSION}/bison-${VERSION}.tgz

# 安装
helm install my-bison bison-${VERSION}.tgz

# 或者创建本地 Helm 仓库
mkdir -p ~/helm-charts
cp bison-*.tgz ~/helm-charts/
helm repo index ~/helm-charts/
helm repo add local-bison ~/helm-charts/
helm install my-bison local-bison/bison --version ${VERSION}
```

## 验证 Package 是否存在于 GHCR

```bash
# 使用 GitHub API 检查
curl -H "Authorization: token YOUR_GITHUB_TOKEN" \
  https://api.github.com/users/SuperMarioYL/packages/container/charts%2Fbison/versions

# 或者尝试拉取（如果是公开的）
helm pull oci://ghcr.io/supermarioyl/charts/bison --version 0.0.7
```

## 检查是否需要认证

即使设置为 public，某些情况下可能仍需要认证:

```bash
# 登录 GHCR
echo YOUR_GITHUB_TOKEN | helm registry login ghcr.io -u SuperMarioYL --password-stdin

# 然后拉取
helm pull oci://ghcr.io/supermarioyl/charts/bison --version 0.0.7
```

## 下一步

1. 先检查 v0.0.7 的 GitHub Actions workflow 日志
2. 确认 Helm chart 推送步骤是否成功
3. 如果推送成功，设置 package 为 public
4. 如果推送失败，发布新版本 v0.0.8 来测试修复后的配置
