# Contributing to KrazyToGo

Thanks for taking an interest. This project stays useful by staying small.

## North star

- **Stability first** — prefer boring fixes over features
- **Tiny surface** — pure `net/http`, no CGO, no frameworks
- **No in-process auth, uploads, WebDAV, or admin UI** — put Tailscale, Caddy, or an Ingress in front
- **Same binary story** — gokrazy, Docker/Podman (`FROM scratch`), and Kubernetes stay first-class

If a change grows the product surface (new flags/env with breaking defaults, new dependencies, or anything that looks like “a little server framework”), open an issue first and expect maintainer review before implementation.

## How to help

1. **Bugs & docs** — high signal. Repro steps, version/commit, and expected vs actual help a lot.
2. **Tests** — preferred before features. Keep them fast and dependency-free.
3. **Security** — follow [`SECURITY.md`](SECURITY.md); do not file public issues for unfixed vulns.

## Dev loop

```bash
go test ./...
go vet ./...
# optional: govulncheck ./...
```

Local smoke:

```bash
mkdir -p data && echo hello > data/hello.txt
go run ./cmd/krazytogo -root ./data -addr :8080
curl -fsS http://127.0.0.1:8080/hello.txt
curl -fsS http://127.0.0.1:8080/healthz
```

## Pull requests

- Keep PRs focused and easy to review
- Do **not** overwrite or “normalize” the existing [`LICENSE`](LICENSE) file
- Match the style of surrounding code; no drive-by refactors
- Update README/examples when user-visible behavior changes

## Support

Enjoying the project? [Buy Me a Coffee](https://buymeacoffee.com/grantgollak).
