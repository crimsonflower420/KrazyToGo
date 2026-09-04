# syntax=docker/dockerfile:1
# Multi-stage build → scratch (Krazy Kontainer). Same static binary as gokrazy.
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY cmd ./cmd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/krazytogo ./cmd/krazytogo

FROM scratch
COPY --from=build /out/krazytogo /krazytogo
# Numeric non-root; volume mounts must be readable by this UID.
USER 65532:65532
ENV KRAZY_ROOT=/data \
    KRAZY_ADDR=:8080 \
    KRAZY_BROWSE=true
EXPOSE 8080
ENTRYPOINT ["/krazytogo"]
