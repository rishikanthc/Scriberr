# Scriberr Electron Shell

This directory contains the macOS desktop shell for Scriberr.

## Local development

From repository root:

```bash
# 1) Build backend binary that Electron launches in dev
go build -o scriberr cmd/server/main.go

# 2) Install desktop shell dependencies
cd desktop/electron
npm install

# 3) Run Electron
npm run dev
```

During startup, Electron now shows a built-in initialization screen while the backend prepares model environments and downloads first-run assets.

By default dev mode looks for backend binary at:

```text
/Users/nico/Developer/quill/scriberr
```

Override backend path if needed:

```bash
SCRIBERR_BACKEND_BIN=/absolute/path/to/scriberr npm run dev
```

## macOS package (DMG)

From `desktop/electron`:

```bash
npm run dist:mac
```

This runs:
- TypeScript compile for Electron main process.
- Frontend build and embed copy into Go backend.
- Go backend build at `dist/desktop-backend/scriberr`.
- Tool bundling at `dist/desktop-tools` (`uv`, `ffmpeg`, `ffprobe`, `yt-dlp`).
- `electron-builder` DMG packaging.

## Tool bundling

By default, `scripts/prepare-desktop-tools.sh` resolves tools from your local `PATH` and copies them into the packaged app resources.
If `yt-dlp` is not found, the script attempts to download the macOS binary automatically.

You can override source paths when building:

```bash
SCRIBERR_UV_SOURCE=/absolute/path/to/uv \
SCRIBERR_FFMPEG_SOURCE=/absolute/path/to/ffmpeg \
SCRIBERR_FFPROBE_SOURCE=/absolute/path/to/ffprobe \
SCRIBERR_YTDLP_SOURCE=/absolute/path/to/yt-dlp \
npm run dist:mac
```
