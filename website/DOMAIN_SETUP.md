# 自定义域名配置指南

本文档说明如何为 Bison 文档站点配置自定义域名 `bison.lei6393.com`。

## 1. DNS 配置

在你的 DNS 服务商（lei6393.com 的域名注册商）添加以下 DNS 记录:

### 方式一：使用 CNAME（推荐）

```
类型: CNAME
主机记录: bison
记录值: supermarioyl.github.io
TTL: 600 (或默认值)
```

### 方式二：使用 A 记录

如果 DNS 服务商不支持 CNAME，可以使用 A 记录指向 GitHub Pages 的 IP:

```
类型: A
主机记录: bison
记录值: 185.199.108.153
TTL: 600

重复添加以下 IP:
185.199.109.153
185.199.110.153
185.199.111.153
```

## 2. GitHub 仓库设置

1. 进入 GitHub 仓库: https://github.com/SuperMarioYL/Bison
2. 点击 **Settings** > **Pages**
3. 在 **Custom domain** 输入框中填写: `bison.lei6393.com`
4. 勾选 **Enforce HTTPS** (DNS 生效后)
5. 点击 **Save**

## 3. 验证配置

### 检查 DNS 解析

```bash
# 检查 CNAME 记录
dig bison.lei6393.com CNAME +short
# 应该返回: supermarioyl.github.io

# 检查 A 记录
dig bison.lei6393.com A +short
# 应该返回 GitHub Pages 的 IP 地址
```

### 测试网站访问

DNS 生效后（通常 5-30 分钟），访问:

- 主域名: https://bison.lei6393.com
- 中文版: https://bison.lei6393.com/zh-Hans/
- 文档: https://bison.lei6393.com/docs/

### 旧 GitHub Pages URL 重定向

GitHub 会自动将以下 URL 重定向到新域名:
- https://supermarioyl.github.io/Bison/ → https://bison.lei6393.com/

## 4. 本地开发

本地开发时仍然使用 `npm start`，会在 `http://localhost:3001/` 运行（注意不再有 `/Bison/` 路径）。

## 5. 部署

自定义域名配置已包含在代码中:

- ✅ `website/static/CNAME` - 包含域名配置
- ✅ `website/docusaurus.config.ts` - URL 和 baseUrl 已更新

每次 `npm run build` 构建时，CNAME 文件会自动复制到 `build/` 目录。

部署到 GitHub Pages:

```bash
cd website
npm run deploy
```

## 常见问题

### Q: DNS 配置后多久生效?
A: 通常 5-30 分钟，最长可能需要 48 小时。

### Q: HTTPS 证书如何配置?
A: GitHub Pages 会在 DNS 生效后自动生成和配置 Let's Encrypt 证书，无需手动操作。

### Q: 为什么选择 bison.lei6393.com 而不是 www.lei6393.com?
A:
- 语义清晰，专门用于 Bison 项目
- 便于未来扩展其他子域名项目
- 符合企业级开源项目的最佳实践

### Q: 旧的 GitHub Pages 链接还能用吗?
A: 可以，GitHub 会自动重定向到新域名。

## 技术细节

当前配置:

```typescript
// docusaurus.config.ts
url: "https://bison.lei6393.com",
baseUrl: "/",
```

这意味着:
- 所有链接使用根路径 `/` 而不是 `/Bison/`
- 中文版访问路径: `/zh-Hans/` (不再是 `/Bison/zh-Hans/`)
- 文档路径: `/docs/` (不再是 `/Bison/docs/`)
