# 🚀 Vercel 自动部署指南

## 第一步：关联 GitHub 仓库

### 1. 访问 Vercel 官网
- 打开 https://vercel.com
- 点击右上角 **"Sign Up"** 或 **"Log In"**
- 使用 GitHub 账号授权登录

### 2. 导入项目
1. 进入 Vercel Dashboard
2. 点击 **"Add New..."** → **"Project"**
3. 选择 **"Import Git Repository"**
4. 输入你的 GitHub 仓库 URL: `https://github.com/layababa/tg_todo`
5. 点击 **"Import"**

---

## 第二步：配置项目设置

### 1. 项目名称
```
Project Name: tg-todo  (或任意名称)
```

### 2. 构建设置
```
Framework Preset: Vite
Root Directory: web
Build Command: npm run build
Output Directory: dist
Install Command: npm install
```

**确认如下配置**：
- ✅ Framework: Vite
- ✅ Root Directory: web
- ✅ Build & Output 与 package.json 匹配

### 3. 环境变量配置

在部署前，点击 **"Environment Variables"**，添加以下变量：

#### 对于所有环境（Production、Preview、Development）：
```
VITE_API_BASE_URL = https://your-backend-api.com
VITE_TELEGRAM_BOT_NAME = @your_bot_name  (可选)
```

**示例填写**：
```
名称: VITE_API_BASE_URL
值:   https://api.yourdomain.com
(或后端服务的实际地址)
```

#### 按环境区分（可选）：
```
# Production 环境
VITE_API_BASE_URL = https://api.yourdomain.com

# Preview 环境
VITE_API_BASE_URL = https://api-staging.yourdomain.com

# Development 环境
VITE_API_BASE_URL = http://localhost:8080
```

---

## 第三步：部署

### 首次部署
1. 配置完成后点击 **"Deploy"** 按钮
2. 等待构建完成（通常 2-5 分钟）
3. 部署成功后会获得一个临时 URL

**部署日志示例**：
```
✓ Installed dependencies
✓ Running "npm run build"
✓ Built in 45s
✓ Deployed to tg-todo.vercel.app
```

### 获取分配的 URL
部署完成后，Vercel 会自动分配：
- **默认 URL**: `https://tg-todo.vercel.app`
- **Git 分支预览**: `https://branch-name.tg-todo.vercel.app`

---

## 第四步：自动部署设置

### ✅ 自动部署（默认启用）

Vercel 默认会监听 GitHub 仓库，每当你推送代码时自动部署：

```bash
# 你的操作
git push origin main

# Vercel 自动：
# 1. 检测到新提交
# 2. 触发构建流程
# 3. 运行 npm run build
# 4. 部署到 https://tg-todo.vercel.app
```

### 配置自动部署规则（可选）

访问 **Project Settings** → **Git** 配置：

```
✓ Production Branch: main
✓ Preview Branches: 所有分支
✓ Deploy on push: 启用
✓ Build on preview deployment: 启用
```

---

## 第五步：自定义域名（可选）

### 添加自定义域名
1. 进入 Project Settings → **Domains**
2. 添加你的域名，例如 `app.yourdomain.com`
3. 按照指示配置 DNS：
   ```
   CNAME: cname.vercel.app
   ```
4. Vercel 会自动生成 SSL 证书

---

## 第六步：监控与调试

### 查看部署日志
- 点击 **"Deployments"** 标签
- 查看最新部署的构建日志
- 出错时会显示错误信息

### 预览环境
- 每个 Pull Request 自动生成预览 URL
- 格式: `https://pr-123.tg-todo.vercel.app`

---

## 常见问题排查

### ❌ 构建失败：找不到模块

**症状**: `Error: Cannot find module '@/...'`

**原因**: 别名配置未正确读取

**解决**:
```bash
# 确保 vite.config.ts 中有别名配置
# 并且 VITE_API_BASE_URL 在 env.d.ts 中声明
```

### ❌ 构建失败：npm 依赖问题

**症状**: `npm ERR! code ETARGET`

**原因**: 依赖版本冲突

**解决**:
```bash
# 本地清理并重新安装
rm package-lock.json
npm install
npm run build
git push
```

### ❌ API 连接失败

**症状**: 前端能打开，但 API 调用失败 (CORS 错误)

**原因**: VITE_API_BASE_URL 配置错误或后端 CORS 未配置

**解决**:
1. 检查环境变量是否正确
2. 确保后端启用了 CORS
3. 在浏览器控制台检查网络错误

### ❌ 长时间未有更新

**症状**: GitHub 有新提交，但 Vercel 没有自动部署

**原因**: Git 连接断开或 Vercel 配置问题

**解决**:
1. 进入 **Project Settings** → **Git**
2. 点击 **"Disconnect Git"**
3. 重新连接 GitHub 仓库

---

## 完整部署流程示例

### 本地开发
```bash
# 1. 开发新功能
git checkout -b feature/new-task

# 2. 测试
npm run dev
npm run build

# 3. 提交代码
git add .
git commit -m "feat: add new task feature"
git push origin feature/new-task
```

### GitHub Pull Request
```
✓ PR 创建
✓ Vercel 自动生成预览 URL
✓ 访问预览检查功能
✓ Merge to main
```

### Vercel 自动部署
```
✓ 检测 main 分支更新
✓ 启动构建 (npm run build)
✓ 部署到生产环境
✓ https://tg-todo.vercel.app 更新
```

---

## 环境变量参考

### 前端环境变量 (.env.local)
```bash
# 本地开发
VITE_API_BASE_URL=http://localhost:8080

# Vercel 部署（在 Vercel Dashboard 中配置）
VITE_API_BASE_URL=https://api.yourdomain.com
```

### Vercel 配置文件 (vercel.json)
```json
{
  "framework": "vite",
  "buildCommand": "cd web && npm run build",
  "outputDirectory": "web/dist",
  "env": {
    "VITE_API_BASE_URL": "@api_base_url"
  }
}
```

---

## 🚀 快速开始命令

```bash
# 1. 推送到 GitHub
git push origin main

# 2. Vercel 自动检测并部署（无需手动操作）

# 3. 检查部署状态
# - 访问 https://vercel.com/dashboard
# - 查看部署日志和预览 URL

# 4. 访问应用
# https://tg-todo.vercel.app
```

---

## ✅ 验证清单

部署后检查以下项：

- [ ] ✅ 应用能正常加载 (https://tg-todo.vercel.app)
- [ ] ✅ API 连接正常 (网络标签页无 CORS 错误)
- [ ] ✅ 任务列表显示正确
- [ ] ✅ 任务操作可用 (完成/删除)
- [ ] ✅ 没有 TypeScript 错误
- [ ] ✅ 没有控制台错误
- [ ] ✅ 响应时间 < 2s

---

## 📞 需要帮助？

**Vercel 官方文档**: https://vercel.com/docs  
**GitHub 集成文档**: https://vercel.com/docs/concepts/git  
**环境变量配置**: https://vercel.com/docs/concepts/projects/environment-variables

