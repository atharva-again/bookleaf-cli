# BookLeaf CLI

Command-line tool for managing the [BookLeaf](https://bookleaf-assignment-atharva.vercel.app) publishing support portal.

## Install

### macOS / Linux

```sh
curl -fsSL https://bookleaf-assignment-atharva.vercel.app/cli/install.sh | bash
```

### Windows

```powershell
irm https://bookleaf-assignment-atharva.vercel.app/cli/install.ps1 | iex
```

### Specific version

```sh
curl -fsSL https://bookleaf-assignment-atharva.vercel.app/cli/install.sh | bash -s -- --version v0.1.6
```

## Quick start

```sh
bookleaf auth login      # Authenticate with your BookLeaf account
bookleaf ticket list     # View your support tickets
bookleaf dashboard       # Show author dashboard
bookleaf --help          # Full command reference
```

## Configuration

The CLI stores configuration at `~/.config/bookleaf/config.json`. The API URL defaults to `https://bookleaf-assignment-atharva.vercel.app` and can be overridden with `bookleaf --api-url <url>` or the `BOOKLEAF_API_URL` environment variable.

## Update

```sh
bookleaf update          # Update to the latest version
bookleaf update v0.1.5   # Update to a specific version
```

## Uninstall

```sh
bookleaf uninstall
```

## Commands

| Command     | Description                            |
|-------------|----------------------------------------|
| `auth`      | Login, logout, and auth status         |
| `ticket`    | List, view, and manage support tickets |
| `book`      | List your books                        |
| `dashboard` | Author dashboard overview              |
| `update`    | Update the CLI binary                  |
| `uninstall` | Remove the CLI and configuration       |
| `whoami`    | Show current user and role             |
