# Security Policy

## Supported versions

KrazyToGo is a small appliance binary. Security fixes land on the latest release
(and `main`). Older tags are not backported unless a release is still marked supported.

## Report a vulnerability

Please report security issues privately via GitHub Security Advisories for
[crimsonflower420/KrazyToGo](https://github.com/crimsonflower420/KrazyToGo),
or email the maintainer listed on the repository.

Do **not** open a public issue for unfixed vulnerabilities.

Include:

- Affected version / commit
- Reproduction steps (local or container is enough)
- Impact (path escape, unexpected disclosure, DoS, etc.)

## Design boundaries (MVP)

- **No in-process auth, uploads, WebDAV, or admin UI.** Put Tailscale, Caddy, Pangolin, or an Ingress in front when you need access control or TLS.
- **Read-only file serving.** Path sanitization rejects `..` and escape-from-root; directory listing is toggleable (`KRAZY_BROWSE` / `-browse`).
- **Static binary:** `CGO_ENABLED=0`, `-ldflags="-s -w"`.
- **Containers:** prefer rootless Podman; never mount the Docker/Podman socket; read-only root filesystem + data volume/PVC only.
- **Kubernetes:** `runAsNonRoot`, `readOnlyRootFilesystem`, `allowPrivilegeEscalation: false`, drop all capabilities, probes on `/healthz`, PVC (not hostPath).

## Supply chain

- CI runs `go test`, `go vet`, and `govulncheck`.
- Prefer pinning GitHub Actions by full commit SHA.
- No secrets in examples — placeholders only.

Thank you for helping keep a tiny file server boring and safe.
