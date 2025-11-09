# syntax=docker/dockerfile:1.6

FROM node:18-alpine AS builder
WORKDIR /app/web

# 先复制依赖文件并安装
COPY web/package*.json ./
RUN npm install

# 复制源码与主题文件
COPY web ./
COPY daisyUI.template.css ../daisyUI.template.css

ARG VITE_API_BASE_URL=http://localhost:8080
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN npm run build

FROM nginx:1.25-alpine AS runner
COPY --from=builder /app/web/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
