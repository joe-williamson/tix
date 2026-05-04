# bg — SRE Breakglass Tickets

Create, link, and approve SREBR tickets in one command.

## Install

Download the latest binary for your OS from [releases][] and put it on your PATH:

```bash
curl -L https://.../bg_linux_amd64.tar.gz | tar xz
sudo mv bg /usr/local/bin/
```

## Setup

You need `~/.jira_config` with your Atlassian API token (you probably already have this):

```ini
[jira]
user_name = your.name@symphony.com
token     = <atlassian-api-token>
```

Generate a token at https://id.atlassian.com/manage-profile/security/api-tokens.

## Use

```bash
bg list                              # see available profiles
bg create c9-prod ESS-46119          # default 12h
bg create prod-cluster ESS-46121 --hours 24
bg info SREBR-20015                  # inspect existing ticket
bg create c9-prod ESS-46119 --dry-run
```

## Personal overrides

To override defaults (e.g. preferred hours), create `~/.bg_profiles.yaml`:

```yaml
defaults:
  hours: 8
```

The team profile list is baked into the binary — to add or change profiles, open a PR in this repo.
