# gokrazy sketch for KrazyToGo

Same static binary; build with `-tags gokrazy` so defaults become `:80` and `/perm/files`.

## Outline

```bash
# Create instance
gok -i krazytogo new

# Add this package (path to your checkout or module), mkfs for /perm, and Tailscale
# Example package lines (edit your gokrazy config / go.mod as appropriate):
#   github.com/crimsonflower420/KrazyToGo/cmd/krazytogo
#   github.com/gokrazy/mkfs
#   plus Tailscale gokrazy packages you already use

# Build with gokrazy tag so main_gokrazy.go applies:
#   go build -tags gokrazy -ldflags="-s -w" ./cmd/krazytogo

gok -i krazytogo edit   # set Tailscale auth key / hostname placeholders only
gok -i krazytogo overwrite --full /dev/sdX   # destructive — pick the right disk
```

## Persistence

- Do **not** store user files on the read-only A/B rootfs.
- Use the fourth partition mounted at `/perm` (ext4 via `github.com/gokrazy/mkfs`).
- Document root default: `/perm/files` (create on first boot if missing — the binary calls `MkdirAll`).
- **Do not use bcachefs.**

## Access

Prefer Tailscale MagicDNS + Serve (or Funnel only if you accept the exposure model).
Do not expose the gokrazy management UI to the public internet.

## Config overrides

Even on gokrazy you can set env or flags if your instance supervision supports them:

| Variable / flag | gokrazy default |
|-----------------|-----------------|
| `KRAZY_ROOT` / `-root` | `/perm/files` |
| `KRAZY_ADDR` / `-addr` | `:80` |
| `KRAZY_BROWSE` / `-browse` | `true` |
