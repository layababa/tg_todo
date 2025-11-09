# Repository Guidelines

## Project Structure & Module Organization
- Product specs (`prd.md`) and front-end requirements (`前端文档.md`) in the repo root define scope; consult them before touching code.
- Keep the Telegram Mini App inside `web/` using Vite + Vue 3: `src/components/` for UI, `src/stores/` for Pinia, `src/services/` for Axios API clients, and `src/assets/` for uiverse.io animations or media.
- Mirror Go back-end work under `server/` with `cmd/bot/` (entry point), `internal/` for domain logic, and `migrations/` for PostgreSQL schema files.
- Reuse the shared palette in `daisyUI.template.css` by importing it into `tailwind.config.ts`; do not redefine colors inline.
- Keep Docker, Compose files, and Vercel configs inside `infra/` so deployment wiring stays isolated from app code.

## Build, Test, and Development Commands
- `cd web && npm install` once to pull Vue/Tailwind/DaisyUI dependencies.
- `cd web && npm run dev` launches the Mini App with Vite and auto-opens the Telegram WebApp bridge.
- `cd web && npm run build` emits the production bundle uploaded to Vercel; run before every PR.
- `cd web && npm run lint` applies ESLint + Stylelint presets; fix issues before committing.
- `cd server && go test ./...` runs all Go unit tests; combine with `docker compose up db` when database access is needed.
- `cd infra && docker compose up --build` brings up Postgres + the Go bot locally to validate bot ↔ Mini App flows.

## Coding Style & Naming Conventions
- Vue files use `<script setup>` with Composition API, two-space indentation, PascalCase component names (`TodoList.vue`), and kebab-case routes (`/todo-detail`).
- Tailwind utility classes stay declarative; for reusable styles, compose them via `@apply` rather than ad-hoc CSS.
- Go code must pass `gofmt` and (optionally) `golangci-lint run`; package names stay short and lowercased (`task`, `auth`).
- Configuration structs, env vars, and DTOs favor descriptive camelCase fields mirroring Telegram payload names.

## Testing Guidelines
- Front-end unit tests live beside components as `*.spec.ts` (Vitest); integration flows sit under `web/tests/e2e` (Cypress) with names like `todo.complete.cy.ts`.
- Snapshot animated components only after stabilizing uiverse.io class names; prefer behavior assertions otherwise.
- Require minimum 80% coverage on `web/src` and enforce with `npm run test -- --coverage`.
- Back-end logic uses table-driven Go tests; mock Telegram HTTP calls with `httptest.Server` and seed Postgres via migration fixtures.

## Commit & Pull Request Guidelines
- Use `type(scope): summary` commit messages (`feat(todo): support batch assign`); group related front/back changes in one commit when they ship together.
- Reference issue or PR IDs in the body (`Refs #12`) and describe any schema or env changes explicitly.
- Pull requests must list test commands executed, link to relevant PRD sections, and include screenshots/GIFs when UI changes touch DaisyUI themes or animations.
- Block PRs without updated docs whenever APIs, theme tokens, or infra scripts change.

## Security & Configuration Tips
- Store Telegram bot tokens, Vercel env secrets, and Postgres credentials in `.env.local` files ignored from Git; provide `.env.example` with safe placeholders.
- Always verify `initData` server-side before trusting user identity, and reject unsigned payloads.
- Regenerate API clients when the Go server contracts change so the Mini App does not hardcode outdated routes.
