# Scriberr macOS Desktop App Plan (Electron Wrapper)

## Decision status

- 2026-02-13: Desktop shell approach confirmed as Electron.
- 2026-02-13: Distribution target confirmed as self-contained (no manual tool installs for end users).

## Current distance to "install and run"

- Completed foundation:
  - Electron shell launches/stops backend and loads app UI.
  - Backend supports env-configurable binary paths for external tools.
  - Packaging flow can bundle `uv`, `ffmpeg`, `ffprobe`, and `yt-dlp` into app resources.
  - WhisperX bootstrap no longer requires `git` (uses ZIP download).
  - Electron startup screen now shows first-run initialization progress from backend logs.
- Remaining work to reach friend-ready release:
  - Validate bundled binaries run on clean Macs (Intel + Apple Silicon).
  - Polish onboarding UX copy and edge-case error messaging.
  - Build signed/notarized DMG and test installation on clean machines.

Estimated remaining effort for friend-ready macOS release:
- Unsigned internal self-contained DMG: 2-4 days.
- Signed/notarized public DMG: +2-4 days.

## Objective

Ship an installable macOS app (`.dmg`) so users launch Scriberr like a native app instead of manually opening `http://localhost:8080` in a browser.

## Why this path

- Current backend is a Go binary that already embeds the React frontend.
- Frontend mostly calls relative API paths (`/api/...`), which works unchanged when loaded from local server URL.
- Wrapping existing architecture is much lower risk than rewriting backend.

## Current constraints (from codebase)

- Backend starts as HTTP server via `cmd/server/main.go`.
- Embedded frontend served from `internal/web/static.go`.
- Runtime tools required by features:
  - `uv` (Python env management for models)
  - `ffmpeg` + `ffprobe` (audio/video operations)
  - `yt-dlp` (YouTube import)
- Auth uses cookies with `Secure` flag tied to `SECURE_COOKIES` config.

## Target architecture (MVP)

1. Electron app launches.
2. Electron starts bundled Go backend as child process with desktop-specific env:
   - `HOST=127.0.0.1`
   - `PORT=<free local port>`
   - `APP_ENV=production`
   - `SECURE_COOKIES=false` (required on local HTTP)
   - `DATABASE_PATH`, `UPLOAD_DIR`, `TRANSCRIPTS_DIR`, `TEMP_DIR`, `WHISPERX_ENV` pointing to user app-data dir.
3. Electron polls `GET /health` until backend is ready.
4. `BrowserWindow` loads `http://127.0.0.1:<port>`.
5. On app quit, Electron terminates backend process gracefully.

## Packaging strategy (macOS)

- Build backend binary for `darwin/arm64` and `darwin/amd64`.
- Package with Electron + `electron-builder`.
- Produce:
  - unsigned `.dmg` for internal QA first
  - signed + notarized `.dmg` for public distribution (phase 2)

## Phased execution plan

## Phase 0: Decisions and acceptance criteria

Deliverables:
- Confirm MVP scope:
  - Intel + Apple Silicon support (separate builds or universal app)
  - unsigned internal build first
  - dependency handling policy (see Phase 2 options)
- Define release acceptance:
  - install app, launch app, register/login, upload audio, run transcription.

## Phase 1: Desktop shell and process orchestration (implementation first slice)

Tasks:
1. Add `desktop/electron/` project (TypeScript):
   - `main.ts` for lifecycle/process management
   - `preload.ts` minimal safe bridge
2. Implement backend process manager:
   - free-port selection
   - child process spawn
   - health-check wait loop with timeout
   - shutdown handling (`before-quit`, `window-all-closed`)
3. Add desktop-specific path bootstrap:
   - use `app.getPath("userData")`
   - create directories `data/uploads`, `data/transcripts`, `data/temp`, `data/whisperx-env`
4. Error UX:
   - if backend fails, show actionable message with logs path.

Acceptance:
- Running `make desktop-dev` launches one app window with working backend and no manual browser steps.

## Phase 2: Build and package pipeline

Tasks:
1. Add root scripts:
   - build frontend
   - copy frontend dist for embed
   - build Go backend for target arch
   - package Electron app
2. Configure `electron-builder`:
   - app id, product name, icons
   - include backend binary in app resources
   - output `.dmg`
3. Add CI workflow for macOS artifact generation.

Acceptance:
- CI and local build can produce installable `.dmg` artifact.

## Phase 3: Dependency strategy and first-run onboarding

Tasks:
1. Bundle runtime tools inside app resources:
   - `uv`
   - `ffmpeg`
   - `ffprobe`
   - `yt-dlp`
2. Inject tool paths into backend env:
   - `SCRIBERR_UV_BIN`
   - `SCRIBERR_FFMPEG_BIN`
   - `SCRIBERR_FFPROBE_BIN`
   - `SCRIBERR_YTDLP_BIN`
3. Keep model/data downloads on first run (acceptable and expected).
4. Add first-run status UX (initializing models/environments).

Acceptance:
- Fresh macOS machine can install and run core workflows without installing external tools manually.
- First-run downloads are visible and understandable in-app.

## Phase 4: Distribution hardening

Tasks:
1. Code signing (Developer ID Application).
2. Notarization and stapling.
3. Update channel setup (optional initially, recommended soon after).

Acceptance:
- Public install on a clean macOS machine without Gatekeeper bypass instructions.

## Risk register

1. Bundled binary compatibility
- Risk: copied binaries may not run on clean target machines.
- Mitigation: validate on clean Intel + Apple Silicon test machines; pin source artifacts.

2. Secure cookie behavior on local HTTP
- Risk: auth/audio playback fails if secure cookies enabled.
- Mitigation: force `SECURE_COOKIES=false` in desktop runtime env.

3. First-run model setup latency
- Risk: app appears broken while downloading models.
- Mitigation: first-run status UI + progress/log surfacing in desktop shell.

4. Apple signing/notarization friction
- Risk: release delays.
- Mitigation: start unsigned QA first, automate notarization once stable.

## Effort estimate (macOS only)

- Phase 1: 2-4 days
- Phase 2: 1-2 days
- Phase 3 (self-contained bundling + onboarding): 2-4 days
- Phase 4 (sign/notarize): 1-2 days
- Total self-contained signed release: ~2 to 3 weeks

## Implementation order after plan approval

1. Build Phase 1 shell + backend orchestration.
2. Package unsigned `.dmg` in Phase 2.
3. Bundle dependencies and first-run UX in Phase 3.
4. Harden and sign in Phase 4.

## Proposed immediate next coding task

Implement first-run onboarding/status view:
- show "initializing model environment" progress
- surface actionable failures from backend logs
- provide retry flow for failed model/tool initialization
