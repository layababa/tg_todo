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

echo "Checking Webhook Info for bot ending in ...${TELEGRAM_BOT_TOKEN: -5}"
response=$(curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getWebhookInfo")
echo "Response: $response"
