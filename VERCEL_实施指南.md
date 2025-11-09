# Vercel 实施指南 - 按步骤执行

## 👤 针对你的项目 (GitHub: layababa/tg_todo)

### 现状分析
- ✅ 代码已推送到 GitHub main 分支
- ✅ 前端编译成功
- ✅ 后端已验证
- ⏳ **待完成**: Vercel 部署配置

---

## 🚀 实施步骤 (按顺序执行)

### 步骤 1: 访问 Vercel 新项目页面 (1 分钟)

**操作**:
1. 在浏览器中打开: https://vercel.com/new
2. 如果没有 Vercel 账户，点击 **"Sign Up"** 用 GitHub 注册
3. 授权 Vercel 访问你的 GitHub 账户

**预期结果**: 看到 "Import Git Repository" 页面

---

### 步骤 2: 导入你的仓库 (1 分钟)

**操作**:
1. 在 "Search your repositories" 搜索框中输入: `tg_todo`
2. 或者直接输入完整 URL: `https://github.com/layababa/tg_todo`
3. 点击 **"Import"** 按钮

**预期结果**: 进入项目配置页面

---

### 步骤 3: 配置构建设置 (2 分钟)

**关键字段** - 按照以下填写:

```
Project Name
└─ tg-todo (或任意名称，Vercel 会自动生成)

Framework Preset
└─ 选择 "Vite"
   (如果自动检测，会有提示，确认即可)

Root Directory
└─ 输入: web
   (重要! 必须是 web 目录)

Build Command
└─ npm run build
   (使用默认值或参考此处)

Output Directory
└─ dist
   (Vite 默认输出目录)

Install Command
└─ npm install
   (默认)
```

**完成**: 点击 **"Deploy"** 按钮

**预期结果**: 
- 看到部署进度条
- 大约 2-5 分钟后完成
- 获得一个临时 URL，如: `https://tg-todo.vercel.app`

---

### 步骤 4: 配置环境变量 (2 分钟)

部署完成后 (或部署过程中):

**操作**:
1. 在 Vercel Dashboard 中找到你的项目 "tg-todo"
2. 点击进入项目
3. 找到 **"Settings"** 标签
4. 点击左侧菜单的 **"Environment Variables"**

**添加变量**:

| Key | Value | 说明 |
|-----|-------|------|
| VITE_API_BASE_URL | `https://your-api.com` | 你的后端 API 地址 |

**具体填写**:
```
Key: VITE_API_BASE_URL
Value: https://api.yourdomain.com
(或任何你的后端 API 真实地址)

或者本地测试时:
Value: http://localhost:8080
```

**完成**: 点击 **"Save"**

**预期结果**: Vercel 自动重新部署

---

### 步骤 5: 验证部署 (2 分钟)

**检查清单**:

```
[ ] 访问 https://tg-todo.vercel.app
    → 页面正常加载

[ ] 打开浏览器开发者工具 (F12)
    → Console 标签页
    → 没有红色错误

[ ] 网络标签页
    → 没有 CORS 错误
    → API 调用成功 (200 状态码)

[ ] 功能测试
    → 任务列表显示
    → 可以完成/删除任务
```

**如果有错误**:
- 查看浏览器控制台错误
- 检查 VITE_API_BASE_URL 是否正确
- 查看 Vercel 部署日志 (Deployments → 最新部署 → Logs)

---

## 🔧 配置后续优化

### 添加自定义域名 (可选)

如果你有自己的域名:

1. 进入项目 Settings → **Domains**
2. 输入域名，如 `app.yourdomain.com`
3. 按照指示配置 DNS (通常是 CNAME 记录)
4. Vercel 会自动生成 SSL 证书

---

### 配置生产 vs 预览环境变量 (可选)

如果你想不同环境使用不同的 API:

```
Environment Variables

VITE_API_BASE_URL:
├─ Production:  https://api.yourdomain.com
├─ Preview:     https://api-staging.yourdomain.com
└─ Development: http://localhost:8080
```

---

## 📊 现在设置的自动化流程

完成上述配置后，你将获得:

```
你的工作流:
1. 编写代码
   ↓
2. git push origin main
   ↓
3. GitHub 通知 Vercel
   ↓
4. Vercel 自动:
   • npm install
   • npm run build
   • 部署到 CDN
   ↓
5. https://tg-todo.vercel.app 自动更新 ✅
   ↓
6. 用户看到最新版本

无需任何人工操作！
```

---

## 🐛 常见问题 - 实时解决

### Q: 部署失败，显示 "Cannot find module"

**A**: 
```bash
# 检查本地是否能构建
cd web
npm install
npm run build

# 如果本地正常，删除 node_modules 后重新提交
rm -rf node_modules package-lock.json
git add .
git commit -m "chore: clean dependencies"
git push origin main
```

---

### Q: 前端能打开，但 API 调用失败 (CORS 错误)

**A**:
```
检查清单:
1. VITE_API_BASE_URL 环境变量是否设置? ✓
2. 值是否正确 (https:// 不是 http://)? ✓
3. 后端是否启用了 CORS 允许 tg-todo.vercel.app? ✓

如果都正确:
• 浏览器强制刷新 (Ctrl+Shift+R)
• 等待 5 分钟后重试
• 查看网络标签页的详细错误
```

---

### Q: 部署卡在 "Building" 状态超过 10 分钟

**A**:
```
通常原因:
• 网络问题 (npm 下载慢)
• 大文件提交 (检查 .gitignore)
• 依赖冲突

解决:
1. 点击 "Cancel" 取消
2. 本地测试 npm run build
3. 删除 node_modules 和 package-lock.json
4. git push 重试
```

---

### Q: 怎样看部署日志?

**A**:
```
Vercel Dashboard
├─ 点击你的项目 "tg-todo"
├─ 找到 "Deployments" 标签
├─ 点击最新的部署
└─ 查看 "Logs" → 完整的构建日志
```

---

### Q: 我修改了代码，但网站没有更新

**A**:
```
检查清单:
1. 代码已提交到 main? git push origin main
2. Vercel 已检测到新提交? (Deployments 显示最新部署)
3. 部署是否成功? (状态不是红色 ❌)
4. 浏览器缓存? (Ctrl+Shift+R 强制刷新)

如果都正确:
• 等待 5 分钟，CDN 缓存可能还在更新
```

---

## ✅ 完成标志

部署成功后，你应该看到:

```
✓ Vercel Dashboard 中有你的项目
✓ 显示最新部署状态: "Ready" (绿色)
✓ 可以访问 https://tg-todo.vercel.app
✓ 能看到任务列表
✓ 浏览器控制台没有错误
✓ 网络请求返回 200 状态码
```

---

## 📞 需要帮助?

如果遇到问题，按此顺序排查:

1. **查看 Vercel 日志**
   - Deployments → 最新部署 → Logs
   
2. **查看浏览器错误**
   - F12 → Console → 查看红色错误
   
3. **检查环境变量**
   - Settings → Environment Variables
   - VITE_API_BASE_URL 是否正确
   
4. **本地测试**
   - cd web && npm run build
   - 确保本地能成功构建

5. **官方文档**
   - https://vercel.com/docs

---

## 🎉 祝贺!

完成这些步骤后，你已经拥有:
- ✅ 完整的前端 CI/CD 流程
- ✅ 自动部署到全球 CDN
- ✅ 自定义域名支持
- ✅ 生产级别的部署基础设施

**现在只需关注代码开发，Vercel 自动处理部署！** 🚀

