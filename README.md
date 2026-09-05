# KrazyToGo

**One tiny static Go binary that serves files — the same build on gokrazy, Docker/Podman (Krazy Kontainer), and Kubernetes.**

[![CI](https://github.com/crimsonflower420/KrazyToGo/actions/workflows/ci.yml/badge.svg)](https://github.com/crimsonflower420/KrazyToGo/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)
[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-Support-yellow?logo=buymeacoffee&logoColor=white)](https://buymeacoffee.com/grantgollak)

Hyper-minimal by design: pure `net/http`, no CGO, no frameworks, and **no** in-process auth, uploads, WebDAV, or admin UI. Put Tailscale, Caddy, or an Ingress in front when you need access control or TLS.

| | |
|---|---|
| **Module** | `github.com/crimsonflower420/KrazyToGo` |
| **Package** | `cmd/krazytogo` |
| **License** | GPL-3.0 (see [`LICENSE`](LICENSE)) |
| **Support** | [Buy Me a Coffee](https://buymeacoffee.com/grantgollak) · GitHub Sponsor button |

## Why

Most “simple” file servers grow a web UI, upload forms, and half an identity stack. KrazyToGo stays boring on purpose:

- **Same binary everywhere** — gokrazy appliance, `FROM scratch` container, or hardened Kubernetes
- **Tiny attack surface** — read-only serving, path escape rejected, directory browse is a flag
- **Ops-friendly** — `/healthz`, Range requests, flags + env with clear precedence
- **Access stays outside** — Tailscale / reverse proxy / Ingress, not baked into the process

## Quick start

Install the binary:

```bash
go install github.com/crimsonflower420/KrazyToGo/cmd/krazytogo@latest
mkdir -p data && echo hello > data/hello.txt
krazytogo -root ./data -addr :8080
# → http://127.0.0.1:8080/hello.txt
# → http://127.0.0.1:8080/healthz  → 200 ok
```

From a checkout (no install):

```bash
mkdir -p data && echo hello > data/hello.txt
go run ./cmd/krazytogo -root ./data -addr :8080
```

Static build:

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o krazytogo ./cmd/krazytogo
./krazytogo -root ./data -addr :8080
```

## Configuration

Precedence: **flags > env > defaults**.

| Flag | Env | Local / container default | gokrazy (`-tags gokrazy`) |
|------|-----|---------------------------|---------------------------|
| `-root` | `KRAZY_ROOT` | `./data` (use `/data` in images) | `/perm/files` |
| `-addr` | `KRAZY_ADDR` | `:8080` | `:80` |
| `-browse` | `KRAZY_BROWSE` | `true` | `true` |

`KRAZY_BROWSE` accepts Go `ParseBool` values (`1`, `t`, `true`, `0`, `f`, `false`, …). When browse is off, directories return 404; files still serve.

## Deploy

### Docker / Podman (Krazy Kontainer)

```bash
docker build -t krazytogo .
docker run -d -p 8080:8080 -v "$PWD/data:/data" -e KRAZY_ROOT=/data krazytogo

podman build -t krazytogo .
podman run -d -p 8080:8080 -v "$PWD/data:/data:Z" -e KRAZY_ROOT=/data krazytogo
```

Compose:

```bash
mkdir -p data
docker compose up --build -d   # or: podman compose up --build -d
curl -fsS http://127.0.0.1:8080/healthz
```

Image is multi-stage → `FROM scratch`, `USER 65532:65532`. Prefer a read-only root filesystem; ensure the data volume is readable by UID 65532.

### Kubernetes

Hardened manifests under [`deploy/kubernetes/`](deploy/kubernetes/) (PVC not hostPath; `runAsNonRoot`; `readOnlyRootFilesystem`; `allowPrivilegeEscalation: false`; drop all caps; `/healthz` probes):

```bash
# Build/push your image, then edit the Deployment image:
kubectl apply -k deploy/kubernetes/
kubectl -n krazytogo rollout status deploy/krazytogo
kubectl -n krazytogo port-forward svc/krazytogo 8080:8080
curl -fsS http://127.0.0.1:8080/healthz
```

### gokrazy

See [`examples/gokrazy-config.md`](examples/gokrazy-config.md). Build with `-tags gokrazy` (listen `:80`, root `/perm/files`). Persist only on `/perm`. Prefer Tailscale for access; do not expose the gokrazy UI publicly.

### Reverse proxy

Example Caddyfile: [`examples/Caddyfile`](examples/Caddyfile). Auth and TLS stay outside this binary.

```mermaid
flowchart LR
  Client --> Edge["Tailscale / Caddy / Ingress"]
  Edge --> KTG["krazytogo\nstatic binary"]
  KTG --> Disk["/data or /perm/files"]
```

## What this is not

Intentionally out of scope: uploads, WebDAV, rich UI, in-process identity, bundling Tailscale into this process, bcachefs.

## Support the project

If KrazyToGo saves you a weekend of yak-shaving, [buy Grant a coffee](https://buymeacoffee.com/grantgollak). The GitHub **Sponsor** button on this repo points at the same page.

## Contributing & security

- See [`CONTRIBUTING.md`](CONTRIBUTING.md) for the tiny-surface rules.
- Security reports: [`SECURITY.md`](SECURITY.md) (private advisories preferred).
- Smoke checklist: `go test ./... && go vet ./...`, Range/`/healthz` checks, browse on/off, container + optional k8s.

## License

[GPL-3.0](LICENSE).
