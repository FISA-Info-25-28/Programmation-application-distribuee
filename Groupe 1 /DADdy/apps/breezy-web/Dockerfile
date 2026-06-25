# syntax=docker/dockerfile:1
# Build context : racine du monorepo (context: . dans compose.yaml)

# ---- deps (cache du npm install — workspace complet) ----
FROM node:22-alpine AS deps
WORKDIR /app
COPY package*.json ./
COPY packages/breezy-shared/package.json ./packages/breezy-shared/
COPY apps/breezy-web/package.json ./apps/breezy-web/
RUN npm install

# ---- dev (hot reload via Vite) ----
FROM node:22-alpine AS dev
WORKDIR /app
ENV CHOKIDAR_USEPOLLING=true
COPY --from=deps /app/node_modules ./node_modules
COPY package*.json ./
COPY packages/breezy-shared ./packages/breezy-shared
COPY apps/breezy-web ./apps/breezy-web
EXPOSE 5173
CMD ["npm", "run", "dev", "--workspace=apps/breezy-web", "--", "--host", "0.0.0.0"]

# ---- build (bundle de prod) ----
FROM deps AS build
COPY packages/breezy-shared ./packages/breezy-shared
COPY apps/breezy-web ./apps/breezy-web
RUN npm run build --workspace=apps/breezy-web

# ---- prod (nginx statique) ----
FROM nginx:1.27-alpine AS prod
COPY --from=build /app/apps/breezy-web/dist /usr/share/nginx/html
COPY apps/breezy-web/nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
