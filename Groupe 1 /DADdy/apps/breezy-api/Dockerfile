# ---- build ----
FROM golang:1.25-alpine AS build
WORKDIR /src

ARG SERVICE

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./${SERVICE}
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/healthcheck ./healthcheck

# ---- dev (hot reload via air) ----
# Utilisé uniquement en local via compose.override.yaml (target: dev).
# Le code source est monté en volume ; air recompile ./$SERVICE à chaque sauvegarde.
FROM golang:1.25-alpine AS dev
WORKDIR /src
# Version pinned (déterministe + layer cachable) au lieu de @latest.
RUN go install github.com/air-verse/air@v1.65.3
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY . .
# SERVICE est fourni au runtime par compose (ex: user-service). Valeur par défaut sûre.
ENV SERVICE=core-api
# Plafonne le parallélisme de compilation (-p) pour éviter la saturation RAM
# quand les services compilent simultanément sur une VM Docker à faible mémoire.
# Coût quasi nul sur les rebuilds chauds (peu d'actions à compiler).
ENV GOFLAGS=-p=2
# air lit .air.toml (qui construit ./$SERVICE) ; le code est monté en volume en dev.
ENTRYPOINT ["air", "-c", "/src/.air.toml"]

# ---- runtime ----
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/server /app/server
COPY --from=build /out/healthcheck /app/healthcheck
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]