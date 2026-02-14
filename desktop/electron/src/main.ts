import { app, BrowserWindow, dialog } from "electron";
import { spawn, type ChildProcessByStdio } from "node:child_process";
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import type { Readable } from "node:stream";

const HEALTH_TIMEOUT_MS = 120_000;
const HEALTH_POLL_INTERVAL_MS = 500;
const STARTUP_LOG_UPDATE_MS = 1_000;

let mainWindow: BrowserWindow | null = null;
let backendProcess: ChildProcessByStdio<null, Readable, Readable> | null = null;
let backendUrl = "";
let backendReady = false;
let isQuitting = false;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function ensureDir(dirPath: string): void {
  fs.mkdirSync(dirPath, { recursive: true });
}

function escapeHtml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

function getDataPaths(): {
  root: string;
  uploads: string;
  transcripts: string;
  temp: string;
  whisperxEnv: string;
  databasePath: string;
  jwtSecretPath: string;
  backendLogPath: string;
} {
  const userDataPath = app.getPath("userData");
  const root = path.join(userDataPath, "data");
  const uploads = path.join(root, "uploads");
  const transcripts = path.join(root, "transcripts");
  const temp = path.join(root, "temp");
  const whisperxEnv = path.join(root, "whisperx-env");
  const databasePath = path.join(root, "scriberr.db");
  const jwtSecretPath = path.join(root, "jwt_secret");

  const logsDir = path.join(userDataPath, "logs");
  const backendLogPath = path.join(logsDir, "backend.log");

  ensureDir(root);
  ensureDir(uploads);
  ensureDir(transcripts);
  ensureDir(temp);
  ensureDir(whisperxEnv);
  ensureDir(logsDir);

  return {
    root,
    uploads,
    transcripts,
    temp,
    whisperxEnv,
    databasePath,
    jwtSecretPath,
    backendLogPath,
  };
}

async function getFreePort(): Promise<number> {
  return new Promise((resolve, reject) => {
    const server = net.createServer();
    server.unref();

    server.on("error", (error) => {
      reject(error);
    });

    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        server.close();
        reject(new Error("Could not allocate a free local port"));
        return;
      }

      const { port } = address;
      server.close((closeErr) => {
        if (closeErr) {
          reject(closeErr);
          return;
        }
        resolve(port);
      });
    });
  });
}

function resolveBackendBinaryPath(): string {
  if (process.env.SCRIBERR_BACKEND_BIN) {
    return process.env.SCRIBERR_BACKEND_BIN;
  }

  if (app.isPackaged) {
    return path.join(process.resourcesPath, "backend", "scriberr");
  }

  return path.resolve(__dirname, "../../../scriberr");
}

function resolveBundledToolPath(toolName: string): string {
  if (app.isPackaged) {
    return path.join(process.resourcesPath, "tools", toolName);
  }
  return path.resolve(__dirname, `../../../dist/desktop-tools/${toolName}`);
}

function getMissingBundledTools(): string[] {
  const toolNames = ["uv", "ffmpeg", "ffprobe", "yt-dlp"];
  const missing: string[] = [];

  for (const name of toolNames) {
    if (!fs.existsSync(resolveBundledToolPath(name))) {
      missing.push(name);
    }
  }

  return missing;
}

function buildBackendEnv(port: number): NodeJS.ProcessEnv {
  const paths = getDataPaths();
  const uvPath = resolveBundledToolPath("uv");
  const ffmpegPath = resolveBundledToolPath("ffmpeg");
  const ffprobePath = resolveBundledToolPath("ffprobe");
  const ytDlpPath = resolveBundledToolPath("yt-dlp");
  const hasBundledTools = fs.existsSync(uvPath) && fs.existsSync(ffmpegPath) && fs.existsSync(ffprobePath) && fs.existsSync(ytDlpPath);

  const env: NodeJS.ProcessEnv = {
    ...process.env,
    HOST: "127.0.0.1",
    PORT: String(port),
    APP_ENV: "production",
    SECURE_COOKIES: "false",
    ALLOWED_ORIGINS: `http://127.0.0.1:${port}`,
    DATABASE_PATH: paths.databasePath,
    JWT_SECRET_FILE: paths.jwtSecretPath,
    UPLOAD_DIR: paths.uploads,
    TRANSCRIPTS_DIR: paths.transcripts,
    TEMP_DIR: paths.temp,
    WHISPERX_ENV: paths.whisperxEnv,
  };

  if (hasBundledTools) {
    env.SCRIBERR_UV_BIN = uvPath;
    env.SCRIBERR_FFMPEG_BIN = ffmpegPath;
    env.SCRIBERR_FFPROBE_BIN = ffprobePath;
    env.SCRIBERR_YTDLP_BIN = ytDlpPath;
  }

  return env;
}

function inferStartupStatus(logTail: string): { title: string; detail: string } {
  const lines = logTail
    .split("\n")
    .map((line) => line.trim())
    .filter((line) => line.length > 0);

  if (lines.length === 0) {
    return {
      title: "Starting Scriberr",
      detail: "Preparing local services...",
    };
  }

  const latest = lines[lines.length - 1];
  const lowered = latest.toLowerCase();

  if (lowered.includes("loading configuration")) {
    return { title: "Loading Configuration", detail: latest };
  }
  if (lowered.includes("connecting to database")) {
    return { title: "Preparing Database", detail: latest };
  }
  if (lowered.includes("setting up authentication")) {
    return { title: "Setting Up Authentication", detail: latest };
  }
  if (lowered.includes("initializing transcription service") || lowered.includes("initializing unified transcription service")) {
    return { title: "Initializing Transcription", detail: latest };
  }
  if (lowered.includes("preparing python environment")) {
    return {
      title: "Preparing AI Environment",
      detail: "Downloading and preparing model runtime. First run can take several minutes.",
    };
  }
  if (lowered.includes("downloading")) {
    return { title: "Downloading Model Assets", detail: latest };
  }
  if (lowered.includes("installing")) {
    return { title: "Installing Runtime Dependencies", detail: latest };
  }
  if (lowered.includes("starting http server")) {
    return { title: "Starting Local Service", detail: latest };
  }
  if (lowered.includes("scriberr is ready")) {
    return { title: "Launching App", detail: "Startup complete." };
  }

  return { title: "Starting Scriberr", detail: latest };
}

function getLogTail(pathToLog: string, maxLines = 80): string {
  if (!fs.existsSync(pathToLog)) {
    return "";
  }
  try {
    const content = fs.readFileSync(pathToLog, "utf8");
    const lines = content.split("\n");
    return lines.slice(-maxLines).join("\n");
  } catch {
    return "";
  }
}

function renderStartupHtml(title: string, detail: string): string {
  return `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>Scriberr</title>
  <style>
    :root {
      color-scheme: dark;
    }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
      background:
        radial-gradient(1200px 700px at 10% 10%, #143447 0%, #0f172a 45%),
        radial-gradient(900px 600px at 90% 90%, #2f2c4a 0%, rgba(15, 23, 42, 0.95) 55%);
      color: #f8fafc;
    }
    .card {
      width: min(700px, 88vw);
      border-radius: 20px;
      padding: 34px 32px;
      background: rgba(15, 23, 42, 0.68);
      border: 1px solid rgba(148, 163, 184, 0.22);
      backdrop-filter: blur(10px);
      box-shadow: 0 18px 40px rgba(2, 6, 23, 0.5);
    }
    .row {
      display: flex;
      align-items: center;
      gap: 16px;
    }
    .spinner {
      width: 20px;
      height: 20px;
      border-radius: 9999px;
      border: 2px solid rgba(148, 163, 184, 0.35);
      border-top-color: #38bdf8;
      animation: spin 1s linear infinite;
      flex: 0 0 auto;
    }
    h1 {
      margin: 0;
      font-size: 22px;
      line-height: 1.2;
      letter-spacing: 0.01em;
      font-weight: 650;
    }
    p {
      margin: 14px 0 0;
      color: #cbd5e1;
      line-height: 1.45;
      font-size: 14px;
      word-break: break-word;
    }
    .hint {
      margin-top: 22px;
      font-size: 12px;
      color: #94a3b8;
    }
    @keyframes spin {
      to { transform: rotate(360deg); }
    }
  </style>
</head>
<body>
  <section class="card">
    <div class="row">
      <div class="spinner" aria-hidden="true"></div>
      <h1>${escapeHtml(title)}</h1>
    </div>
    <p>${escapeHtml(detail)}</p>
    <p class="hint">Scriberr is preparing local AI models and services. First launch can take several minutes.</p>
  </section>
</body>
</html>`;
}

async function loadStartupScreen(window: BrowserWindow, title: string, detail: string): Promise<void> {
  const html = renderStartupHtml(title, detail);
  await window.loadURL(`data:text/html;charset=utf-8,${encodeURIComponent(html)}`);
}

function attachBackendLogs(processHandle: ChildProcessByStdio<null, Readable, Readable>): void {
  const { backendLogPath } = getDataPaths();
  const logStream = fs.createWriteStream(backendLogPath, { flags: "a" });
  const stamp = new Date().toISOString();
  logStream.write(`\n[${stamp}] Starting Scriberr backend\n`);

  processHandle.stdout.on("data", (chunk: Buffer) => {
    logStream.write(chunk);
  });
  processHandle.stderr.on("data", (chunk: Buffer) => {
    logStream.write(chunk);
  });
  processHandle.on("exit", (code, signal) => {
    const exitStamp = new Date().toISOString();
    logStream.write(`[${exitStamp}] Backend exited code=${String(code)} signal=${String(signal)}\n`);
    logStream.end();
  });
}

function startBackend(port: number): void {
  const backendBinaryPath = resolveBackendBinaryPath();
  if (!fs.existsSync(backendBinaryPath)) {
    throw new Error(
      `Backend binary not found at ${backendBinaryPath}. Build it with 'go build -o scriberr cmd/server/main.go'.`,
    );
  }

  const env = buildBackendEnv(port);
  const childProcess = spawn(backendBinaryPath, [], {
    cwd: path.dirname(backendBinaryPath),
    env,
    stdio: ["ignore", "pipe", "pipe"],
  });

  backendProcess = childProcess;
  attachBackendLogs(childProcess);

  childProcess.on("error", (error) => {
    if (isQuitting) {
      return;
    }
    dialog.showErrorBox("Scriberr Backend Error", `Failed to start backend process:\n${error.message}`);
  });

  childProcess.on("exit", (code, signal) => {
    if (isQuitting) {
      return;
    }

    dialog.showErrorBox(
      "Scriberr Backend Stopped",
      [
        `The backend process exited unexpectedly.`,
        `Exit code: ${String(code)}, signal: ${String(signal)}.`,
        `Logs: ${getDataPaths().backendLogPath}`,
      ].join("\n"),
    );
  });
}

async function waitForBackendHealthy(url: string): Promise<void> {
  const startedAt = Date.now();
  let lastError = "";

  while (Date.now() - startedAt < HEALTH_TIMEOUT_MS) {
    try {
      const response = await fetch(`${url}/health`, { cache: "no-store" });
      if (response.ok) {
        return;
      }
      lastError = `Unexpected status: ${response.status}`;
    } catch (error) {
      lastError = error instanceof Error ? error.message : String(error);
    }

    await sleep(HEALTH_POLL_INTERVAL_MS);
  }

  throw new Error(`Backend did not become healthy within ${HEALTH_TIMEOUT_MS}ms. Last error: ${lastError}`);
}

function createMainWindow(): BrowserWindow {
  const window = new BrowserWindow({
    width: 1360,
    height: 900,
    minWidth: 1080,
    minHeight: 760,
    webPreferences: {
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, "preload.js"),
    },
  });

  window.on("closed", () => {
    mainWindow = null;
  });

  return window;
}

async function stopBackend(): Promise<void> {
  if (!backendProcess) {
    backendReady = false;
    return;
  }

  const processHandle = backendProcess;
  backendProcess = null;
  backendReady = false;

  if (processHandle.killed) {
    return;
  }

  await new Promise<void>((resolve) => {
    const killTimer = setTimeout(() => {
      processHandle.kill("SIGKILL");
      resolve();
    }, 8_000);

    processHandle.once("exit", () => {
      clearTimeout(killTimer);
      resolve();
    });

    processHandle.kill("SIGTERM");
  });
}

async function boot(): Promise<void> {
  backendReady = false;
  mainWindow = createMainWindow();
  await loadStartupScreen(mainWindow, "Starting Scriberr", "Preparing local services...");

  if (app.isPackaged) {
    const missingTools = getMissingBundledTools();
    if (missingTools.length > 0) {
      throw new Error(`Packaged tool bundle is incomplete. Missing: ${missingTools.join(", ")}`);
    }
  }

  const port = await getFreePort();
  backendUrl = `http://127.0.0.1:${port}`;

  startBackend(port);
  const backendLogPath = getDataPaths().backendLogPath;
  let lastStartupSignature = "";

  const updateTimer = setInterval(() => {
    if (!mainWindow || mainWindow.isDestroyed()) {
      return;
    }
    const tail = getLogTail(backendLogPath);
    const { title, detail } = inferStartupStatus(tail);
    const signature = `${title}::${detail}`;
    if (signature === lastStartupSignature) {
      return;
    }
    lastStartupSignature = signature;
    void loadStartupScreen(mainWindow, title, detail);
  }, STARTUP_LOG_UPDATE_MS);

  try {
    await waitForBackendHealthy(backendUrl);
    backendReady = true;
  } finally {
    clearInterval(updateTimer);
  }

  if (!mainWindow || mainWindow.isDestroyed()) {
    return;
  }
  await mainWindow.loadURL(backendUrl);
}

app.on("before-quit", () => {
  isQuitting = true;
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") {
    app.quit();
  }
});

app.on("activate", async () => {
  if (BrowserWindow.getAllWindows().length !== 0) {
    return;
  }

  if (backendReady && backendUrl) {
    mainWindow = createMainWindow();
    await mainWindow.loadURL(backendUrl);
    return;
  }

  mainWindow = createMainWindow();
  const tail = getLogTail(getDataPaths().backendLogPath);
  const { title, detail } = inferStartupStatus(tail);
  await loadStartupScreen(mainWindow, title, detail);
});

void app.whenReady()
  .then(async () => {
    try {
      await boot();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      const backendPath = resolveBackendBinaryPath();
      const logPath = getDataPaths().backendLogPath;
      const detail = [
        message,
        "",
        `Backend binary path: ${backendPath}`,
        `Backend logs: ${logPath}`,
        "",
        "Required runtime tools must also be available: uv, ffmpeg, ffprobe, yt-dlp.",
        "Packaged builds should include these under app resources/tools.",
      ].join("\n");

      const buttonIndex = dialog.showMessageBoxSync({
        type: "error",
        title: "Scriberr Startup Failed",
        message: "Scriberr could not start.",
        detail,
        buttons: ["Quit", "Retry"],
        defaultId: 1,
        cancelId: 0,
      });

      if (buttonIndex === 1) {
        app.relaunch();
        app.exit(0);
        return;
      }

      app.exit(1);
    }
  })
  .catch((error) => {
    const message = error instanceof Error ? error.message : String(error);
    dialog.showErrorBox("Scriberr Fatal Error", message);
    app.exit(1);
  });

process.on("SIGINT", async () => {
  isQuitting = true;
  await stopBackend();
  app.exit(0);
});

process.on("SIGTERM", async () => {
  isQuitting = true;
  await stopBackend();
  app.exit(0);
});

app.on("will-quit", async (event) => {
  event.preventDefault();
  await stopBackend();
  app.exit(0);
});
