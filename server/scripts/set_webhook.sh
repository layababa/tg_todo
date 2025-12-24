#!/bin/bash

# Telegram Webhook 设置脚本
# 用途：将 Bot 的 Webhook 指向你的服务器

set -e

# 从 .env 读取配置
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

BOT_TOKEN="${TELEGRAM_BOT_TOKEN}"
WEBHOOK_URL="${TELEGRAM_WEBHOOK_URL}"

if [ -z "$BOT_TOKEN" ]; then
    echo "❌ 错误：未找到 TELEGRAM_BOT_TOKEN"
    echo "请在 .env 文件中设置 TELEGRAM_BOT_TOKEN"
    exit 1
fi

if [ -z "$WEBHOOK_URL" ]; then
    echo "❌ 错误：未找到 TELEGRAM_WEBHOOK_URL"
    echo "请在 .env 文件中设置 TELEGRAM_WEBHOOK_URL"
    echo "示例：TELEGRAM_WEBHOOK_URL=https://your-domain.com/webhook/telegram"
    exit 1
fi

echo "🔧 设置 Telegram Webhook..."
echo "Bot Token: ${BOT_TOKEN:0:10}..."
echo "Webhook URL: $WEBHOOK_URL"
echo ""

# 调用 Telegram API 设置 Webhook
RESPONSE=$(curl -s -X POST "https://api.telegram.org/bot${BOT_TOKEN}/setWebhook" \
    -H "Content-Type: application/json" \
    -d "{\"url\": \"${WEBHOOK_URL}\"}")

echo "📡 Telegram API 响应："
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# 检查是否成功
if echo "$RESPONSE" | grep -q '"ok":true'; then
    echo "✅ Webhook 设置成功！"
    echo ""
    echo "📋 验证 Webhook 状态："
    curl -s "https://api.telegram.org/bot${BOT_TOKEN}/getWebhookInfo" | jq '.'
else
    echo "❌ Webhook 设置失败"
    exit 1
fi
