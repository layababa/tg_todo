# ⚡ Vercel 部署 - 快速开始 (5 分钟)

## 🎯 3 个核心步骤

### 步骤 1️⃣：GitHub 授权关联 (1 分钟)

```
1. 打开 https://vercel.com/new
2. 点击 "Import Git Repository"
3. 输入: https://github.com/layababa/tg_todo
4. 点击 "Import"
```

**授权 GitHub 账号**（首次需要）

---

### 步骤 2️⃣：配置构建参数 (2 分钟)

在 Vercel 部署向导中填入：

| 配置项 | 值 |
|--------|-----|
| Framework | Vite |
| Root Directory | web |
| Build Command | npm run build |
| Output Directory | dist |

**点击 "Deploy"** ⚡

---

### 步骤 3️⃣：配置环境变量 (2 分钟)

部署后，进入 **Settings** → **Environment Variables**

添加：
```
Key:   VITE_API_BASE_URL
Value: https://your-api-domain.com
```

**点击 "Save"**，自动重新部署 ✅

---

## 🚀 现在每次推送代码会自动部署！

```bash
# 你只需要这样做：
git push origin main

# Vercel 会自动：
# 1. 检测新提交
# 2. 运行 npm run build
# 3. 部署到 https://tg-todo.vercel.app
```

---

## 📊 查看部署状态

### Vercel Dashboard
```
https://vercel.com/dashboard
→ 选择 "tg-todo" 项目
→ 查看 "Deployments" 标签
```

### 实时部署日志
```
每次 git push 后
→ Vercel 显示构建进度
→ 成功/失败提示
```

---

## 🔗 你的应用地址

部署成功后，访问：
```
https://tg-todo.vercel.app
(或自定义域名)
```

---

## ❌ 常见错误 - 5 秒解决

### 错误 1: "Cannot find module"
```bash
# 原因: 依赖未安装
# 解决:
rm package-lock.json
npm install
git push
```

### 错误 2: "CORS error"
```
原因: API 地址错误
解决: 检查 Settings → Environment Variables
     VITE_API_BASE_URL 是否正确
```

### 错误 3: 没有自动部署
```
原因: Git 连接断开
解决: Settings → Git → Reconnect Repository
```

---

## 🌍 预览每个分支

```bash
# 创建功能分支
git checkout -b feature/new

# 推送
git push origin feature/new

# Vercel 自动生成预览 URL:
# https://feature-new.tg-todo.vercel.app

# PR 也会自动关联预览链接
```

---

## ✨ 就这么简单！

| 步骤 | 时间 | 说明 |
|------|------|------|
| 1️⃣ 导入仓库 | 1 分钟 | https://vercel.com/new |
| 2️⃣ 配置构建 | 2 分钟 | Framework: Vite, Root: web |
| 3️⃣ 配置环境变量 | 2 分钟 | VITE_API_BASE_URL |
| ✅ 完成 | 5 分钟 | 自动部署已启用 |

---

## 📞 需要帮助？

**官方文档**: https://vercel.com/docs/concepts/git  
**YouTube 教程**: https://www.youtube.com/results?search_query=vercel+deploy  
**中文教程**: 搜索 "Vercel GitHub 自动部署"

