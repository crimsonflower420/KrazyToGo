# KrazyToGo

Tiny Go file server. One static binary on gokrazy, Docker/Podman (Krazy Kontainer), and Kubernetes.

Hyper-minimal: pure `net/http`, no CGO, no frameworks, no in-process auth/uploads/WebDAV/UI. Put Tailscale, Caddy, or an Ingress in front when you need access control.

**Module:** `github.com/crimsonflower420/KrazyToGo`  
**Package:** `cmd/krazytogo`

> **LICENSE:** The public GitHub repository already has a `LICENSE` file. Do not overwrite it when syncing this tree — push application files only and keep the existing license.

## Quick start (local)

```bash
mkdir -p data
echo hello > data/hello.txt
go run ./cmd/krazytogo -root ./data -addr :8080
# → http://127.0.0.1:8080/hello.txt
# → http://127.0.0.1:8080/healthz  → 200 ok
```

Static build:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o krazytogo ./cmd/krazytogo
./krazytogo -root ./data -addr :8080
```

## Configuration

Precedence: **flags > env > defaults**.

| Flag | Env | Local/container default | gokrazy default (`-tags gokrazy`) |
|------|-----|-------------------------|-----------------------------------|
| `-root` | `KRAZY_ROOT` | `./data` (set `/data` in images) | `/perm/files` |
| `-addr` | `KRAZY_ADDR` | `:8080` | `:80` |
| `-browse` | `KRAZY_BROWSE` | `true` | `true` |

`KRAZY_BROWSE` accepts Go `ParseBool` values (`1`, `t`, `true`, `0`, `f`, `false`, …). When browse is off, directories return 404; files still serve.

## Docker / Podman (Krazy Kontainer)

```bash
docker build -t krazytogo .
docker run -d -p 8080:8080 -v "$PWD/data:/data" -e KRAZY_ROOT=/data krazytogo

podman build -t krazytogo .
podman run -d -p 8080:8080 -v "$PWD/data:/data:Z" -e KRAZY_ROOT=/data krazytogo
```

Compose (Docker or Podman):

```bash
mkdir -p data
docker compose up --build -d
# or: podman compose up --build -d
curl -fsS http://127.0.0.1:8080/healthz
```

Image is multi-stage → `FROM scratch`, `USER 65532:65532`, read-only root recommended. Ensure the data volume is readable/writable by UID 65532.

## Kubernetes

Hardened manifests under `deploy/kubernetes/` (PVC, not hostPath; `runAsNonRoot`; `readOnlyRootFilesystem`; `allowPrivilegeEscalation: false`; drop all caps; `/healthz` probes):

```bash
# Build/push image to your registry, then edit deployment image:
kubectl apply -k deploy/kubernetes/
kubectl -n krazytogo rollout status deploy/krazytogo
kubectl -n krazytogo port-forward svc/krazytogo 8080:8080
curl -fsS http://127.0.0.1:8080/healthz
```

## gokrazy

See [examples/gokrazy-config.md](examples/gokrazy-config.md). Build with `-tags gokrazy` so listen `:80` and root `/perm/files`. Persist only on `/perm` (ext4 via `gokrazy/mkfs`). Prefer Tailscale for access; do not expose the gokrazy UI publicly.

## Reverse proxy

Example Caddyfile: [examples/Caddyfile](examples/Caddyfile). Auth and TLS stay outside this binary.

## Smoke checklist

1. `go test ./... && go vet ./...`
2. Local: serve a fixture file; `Range: bytes=0-3` returns 206; `/healthz` is `ok`.
3. Browse on: directory listing HTML. Browse off: directory 404, file still 200.
4. Container: Docker and rootless Podman with `:Z` on SELinux hosts.
5. Optional: `kubectl apply -k deploy/kubernetes/` against a disposable cluster.

## Out of scope (intentionally)

Uploads, WebDAV, rich UI, in-process identity, bundling Tailscale into this process, bcachefs.

## Security

See [SECURITY.md](SECURITY.md).
