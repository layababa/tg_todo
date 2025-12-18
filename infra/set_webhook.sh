#!/bin/bash
# Load env vars safely
while IFS='=' read -r key value; do
  if [[ $key =~ ^[^#]*$ ]] && [[ -n $key ]]; then
    export "$key=$value"
  fi
done < ../server/.env

if [ -z "$TELEGRAM_BOT_TOKEN" ]; then
    echo "Error: TELEGRAM_BOT_TOKEN not found"
    exit 1
fi

WEBHOOK_URL="https://ddddapp.zcvyzest.xyz/webhook/telegram"
echo "Setting Webhook to $WEBHOOK_URL with secret token..."

curl -s -X POST "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/setWebhook" \
     -d "url=$WEBHOOK_URL" | json_pp
