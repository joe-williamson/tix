# tix — SRE Breakglass Tickets

Create, link, and approve SREBR tickets in one command.

## Build

```bash
git clone https://github.com/joe-williamson/tix.git
cd tix
make build        # produces ./tix binary
make install      # installs to $GOPATH/bin
```

Requires Go 1.24.1+ (Go 1.24.0 has a known compiler bug — upgrade with `go install golang.org/dl/go1.24.5@latest && go1.24.5 download`).

## Setup

### 1. Jira credentials

`tix` reads `~/.jira_config` automatically — no path configuration needed. Create it if you don't have it:

```ini
[jira]
user_name = your.name@example.com
token     = <atlassian-api-token>
```

Generate a token at https://id.atlassian.com/manage-profile/security/api-tokens.

### 2. Breakglass profiles

Copy the team profiles file to `~/.bg_profiles.yaml`:

```bash
cp bg_profiles.yaml ~/.bg_profiles.yaml
```

Edit `defaults.user` to your username.

## Use

```bash
tix list                                    # see available profiles
tix bg c9-prod ESS-46119                    # create breakglass ticket (default 12h)
tix bg prod-cluster ESS-46121 --hours 24   # override hours
tix bg tlm-prod ESS-46120 --dry-run        # preview without creating
tix info ESS-46119                          # inspect a ticket
tix info ESS-46119 --comments              # include comments
```

## Environment overrides

| Variable | Default | Purpose |
|---|---|---|
| `JIRA_URL` | `https://perzoinc.atlassian.net` | Override Jira instance |
| `BG_PROFILES` | `~/.bg_profiles.yaml` | Override profiles file path |
