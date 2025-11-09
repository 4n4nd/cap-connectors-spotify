# connectors-spotify (starter)

A thin adapter service that talks to the Spotify Web API for the **Custom Public Music Playlist** platform.

**Status:** Starter scaffold with working HTTP server, routes, and typed client stubs. Replace stub logic with real Spotify calls and OAuth storage.

## What it does (v0)
- Exposes internal HTTP endpoints your control-plane can call:
  - `POST /v1/spotify/devices:list` → list devices for a user
  - `POST /v1/spotify/playback:transfer` → transfer playback to a device
  - `POST /v1/spotify/queue:append` → add a track to the queue
  - `GET  /v1/spotify/tracks:search?q=&limit=&market=` → search tracks
  - `GET  /v1/spotify/tracks:by-isrc?isrc=` → resolve a track by ISRC
- Implements structure for token handling & rate limiting (stubs), JSON logging, graceful shutdown.

## Quick start

```bash
git clone https://github.com/4n4nd/cap-connectors-spotify.git
cd connectors-spotify

cp .env.example .env
make tidy
make run
# in another shell
curl -s localhost:8081/healthz
```

## Env

| var | required | default | note |
|-----|----------|---------|------|
| `HTTP_PORT` | no | `8081` | service port |
| `LOG_LEVEL` | no | `info` | `debug|info|warn|error` |
| `SPOTIFY_CLIENT_ID` | yes (for OAuth) |  | app client id |
| `SPOTIFY_CLIENT_SECRET` | yes (for OAuth) |  | app secret (avoid if using PKCE) |
| `SPOTIFY_REDIRECT_URI` | yes |  | OAuth redirect URL |
| `TOKEN_ENC_KEY` | no |  | base64 key for token encryption (to implement) |

## Build & Dev

```bash
make run      # run locally
make build    # build ./bin/conn-spotify
make docker   # build container image
make test     # run unit tests (sparse for now)
```

## Notes

- This starter uses **chi** for routing and **zerolog** for JSON logs.
- Token store is in-memory; replace with Postgres/Redis for real use.
- Rate limiting/backoff logic is minimal; expand as needed.
- Spotify Web API requires user OAuth tokens for playback control.
