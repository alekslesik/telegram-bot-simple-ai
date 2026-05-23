# AGENTS.md

## Cursor Cloud specific instructions

### Overview

Go 1.26.2 Telegram bot (`telegram-bot-simple`) with PostgreSQL 17. Standard commands are in the `Makefile`; see `make help` for a full list.

### Services

| Service | How to start | Notes |
|---------|-------------|-------|
| PostgreSQL 17 | `docker compose up -d postgres` | Required at runtime; exposes port 5432 |
| Bot | `make run` (loads `.env` automatically) | Requires valid `TOKEN` in `.env` |

### Running lint / test / build

- **Lint:** `make lint` (runs `fmt`, `vet`, `staticcheck`, `golangci-lint`)
- **Test:** `make test`
- **Build:** `make build`
- **Full preprod pipeline:** `make preprod` (deps → fmt → imports → vet → staticcheck → golangci-lint → test → vuln → docker-build)

### Non-obvious caveats

- `$HOME/go/bin` must be in `PATH` for `golangci-lint` to be found by the Makefile. The update script adds this to `~/.bashrc`.
- `golangci-lint` must be built with Go 1.26+ (`GOTOOLCHAIN=$(go env GOVERSION) go install ...`); otherwise it errors on the module's Go version.
- `govulncheck` (`make vuln`) may exit non-zero due to Go stdlib vulnerabilities (not project code). This is expected until Go is patched upstream.
- The `.env.example` file has Windows-style line endings (`\r\n`). Do **not** source `.env` directly via `. ./.env`; instead pass variables individually or use `make run` which handles this. Alternatively, run `sed -i 's/\r$//' .env` after copying.
- Docker daemon must be running before `make docker-build` or `make docker-compose-up`. In Cloud Agent VMs, start it with `sudo dockerd &>/tmp/dockerd.log &` and configure `fuse-overlayfs` + `iptables-legacy` (see nested-container instructions).
- The bot requires a valid Telegram bot token (`TOKEN` env var from `@BotFather`). Without it, the bot starts, connects to PostgreSQL, probes the Telegram API network, but fails Telegram authentication after 5 retries.
