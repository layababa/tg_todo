# syntax=docker/dockerfile:1.6

FROM node:20-alpine AS builder
WORKDIR /app

# Install pnpm
RUN npm install -g pnpm

# Copy project files
COPY web/package.json web/package-lock.json* ./web/
COPY daisyUI.template.css ./

# Install dependencies
WORKDIR /app/web
RUN pnpm install

# Copy source
COPY web ./

ARG VITE_API_BASE_URL=https://api.xwqpfzmlj.com
ENV VITE_API_BASE_URL=${VITE_API_BASE_URL}

RUN pnpm run build

FROM nginx:1.25-alpine AS runner
COPY --from=builder /app/web/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
