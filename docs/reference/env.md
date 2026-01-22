# Environment variables

This reference lists the supported environment variables for the API and frontend.

## Backend (`services/api`)
Values are loaded from `.env` in the current directory or parent directories
before reading process env. Copy from `services/api/.env.example` to start.

- `APP_ENV`
  - `local` enables local-friendly defaults.
- `PORT`
  - Default: `8080`.
- `DATABASE_URL`
  - Default: `postgres://ultimate_ticket:ultimate_ticket@localhost:5432/ultimate_ticket?sslmode=disable`.
- `CORS_ORIGINS`
  - Default: `http://localhost:5173,http://127.0.0.1:5173`.
- `SESSION_TTL`
  - Default: `1h`.
- `ADMIN_USERNAME`
  - Used by `authctl bootstrap-admin` and `authctl reset-auth`.
- `ADMIN_PASSWORD`
  - Used by `authctl bootstrap-admin` and `authctl reset-auth`.
- `ADMIN_EMAIL`
  - Used by `authctl bootstrap-admin` and `authctl reset-auth`.
- `SESSION_COOKIE_SECURE`
  - Default: `false` when `APP_ENV=local`, `true` otherwise.
- `ALLOW_PUBLIC_REGISTER`
  - Default: `true` when `APP_ENV=local`, `false` otherwise.
- `SERVICE_NAME`
  - Default: `api`; used in request logs.

## Frontend (`frontend`)
Copy from `frontend/.env.example` to start.

- `VITE_API_BASE_URL`
  - Default: `http://localhost:8080`.
- `FRONTEND_PORT`
  - Default: `5173`.
