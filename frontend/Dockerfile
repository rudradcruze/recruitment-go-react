# syntax=docker/dockerfile:1
# The build context is the root of the repo, because the pnpm workspace is there. See compose.yaml.

FROM node:24-alpine AS build
RUN npm install --global pnpm@12.15.0
WORKDIR /repo
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY frontend/package.json frontend/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store pnpm install --frozen-lockfile
COPY frontend frontend
# The demo Admin of the login page. Vite puts these values in the files of the page, so they are public.
# Leave them empty in deployment, and the page shows no demo Admin.
ARG VITE_DEMO_ADMIN_EMAIL=
ARG VITE_DEMO_ADMIN_PASSWORD=
ENV VITE_DEMO_ADMIN_EMAIL=$VITE_DEMO_ADMIN_EMAIL VITE_DEMO_ADMIN_PASSWORD=$VITE_DEMO_ADMIN_PASSWORD
RUN pnpm --filter frontend run build

# nginx serves the frontend, and sends /api, /docs and /openapi.yaml to the API.
# It listens on port 8080, because a process without special rights cannot use port 80. compose.yaml publishes it on 5651.
FROM nginxinc/nginx-unprivileged:1.29-alpine
COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY frontend/nginx-security-headers.conf /etc/nginx/security-headers.conf
COPY --from=build /repo/frontend/dist /usr/share/nginx/html
EXPOSE 8080
