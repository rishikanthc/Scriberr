import { useCallback, useEffect, useRef, useState } from "react";
import type { RecordingSession } from "@/features/recording/api/recordingsApi";
import {
  useCancelRecording,
  useCreateRecording,
  useStopRecording,
  useUploadRecordingChunk,
} from "@/features/recording/hooks/useRecordingSession";
import {
  buildMicrophoneConstraints,
  getBrowserRecordingSupport,
  mediaRecorderOptionsFor,
  selectRecordingMimeType,
  type BrowserRecordingSupport,
  type MicrophoneConstraintSelection,
  type RecordingMimeSelection,
} from "@/features/recording/utils/mediaRecorderSupport";
import {
  createRecordingDurationState,
  elapsedRecordingDurationMs,
  pauseRecordingDuration,
  resumeRecordingDuration,
  stopRecordingDuration,
  type RecordingDurationState,
} from "@/features/recording/utils/recordingDuration";

export type BrowserRecorderStatus =
  | "idle"
  | "unsupported"
  | "permission-denied"
  | "permission-ready"
  | "recording"
  | "paused"
  | "stopping"
  | "finalizing"
  | "ready"
  | "failed"
  | "canceled";

export type BrowserRecorderStartOptions = {
  title: string;
  deviceId?: string;
  chunkDurationMs?: number;
  autoTranscribe?: boolean;
  profileId?: string;
  language?: string;
  diarization?: boolean;
};

export type RecordingInputDevice = {
  deviceId: string;
  label: string;
  groupId?: string;
};

export type BrowserRecorderState = {
  status: BrowserRecorderStatus;
  session: RecordingSession | null;
  error: string | null;
  elapsedMs: number;
  selectedMimeType: string;
  appliedSettings: MediaTrackSettings | null;
  requestedConstraints: MicrophoneConstraintSelection["requested"] | null;
  pendingChunks: number;
  uploadedChunks: number;
  failedChunkIndex: number | null;
  availableDevices: RecordingInputDevice[];
  selectedDeviceId: string;
  devicesLoading: boolean;
  devicesError: string | null;
};

type PendingChunk = {
  index: number;
  blob: Blob;
  mimeType: string;
  durationMs?: number;
};

const defaultChunkDurationMs = 5000;

export function useBrowserRecorder() {
  const createRecordingMutation = useCreateRecording();
  const uploadChunkMutation = useUploadRecordingChunk();
  const stopRecordingMutation = useStopRecording();
  const cancelRecordingMutation = useCancelRecording();

  const [state, setState] = useState<BrowserRecorderState>(() => ({
    status: getBrowserRecordingSupport().supported ? "idle" : "unsupported",
    session: null,
    error: supportErrorMessage(getBrowserRecordingSupport()),
    elapsedMs: 0,
    selectedMimeType: "",
    appliedSettings: null,
    requestedConstraints: null,
    pendingChunks: 0,
    uploadedChunks: 0,
    failedChunkIndex: null,
    availableDevices: [],
    selectedDeviceId: "",
    devicesLoading: false,
    devicesError: null,
  }));

  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const sessionRef = useRef<RecordingSession | null>(null);
  const chunkIndexRef = useRef(0);
  const chunkStartedAtRef = useRef<number | null>(null);
  const durationRef = useRef<RecordingDurationState | null>(null);
  const uploadChainRef = useRef(Promise.resolve());
  const failedChunksRef = useRef<PendingChunk[]>([]);
  const stopResolverRef = useRef<(() => void) | null>(null);
  const acceptingChunksRef = useRef(false);

  const setError = useCallback((error: unknown, fallback: string) => {
    setState((current) => ({
      ...current,
      status: isPermissionError(error) ? "permission-denied" : "failed",
      error: error instanceof Error ? error.message : fallback,
    }));
  }, []);

  const stopStream = useCallback(() => {
    streamRef.current?.getTracks().forEach((track) => track.stop());
    streamRef.current = null;
  }, []);

  const refreshInputDevices = useCallback(async () => {
    if (!navigator.mediaDevices?.enumerateDevices) {
      setState((current) => ({
        ...current,
        availableDevices: [],
        selectedDeviceId: "",
        devicesLoading: false,
        devicesError: "This browser cannot list microphone devices.",
      }));
      return;
    }

    setState((current) => ({
      ...current,
      devicesLoading: true,
      devicesError: null,
    }));

    try {
      const mediaDevices = await navigator.mediaDevices.enumerateDevices();
      const inputDevices = toRecordingInputDevices(mediaDevices);
      setState((current) => ({
        ...current,
        availableDevices: inputDevices,
        selectedDeviceId: selectedInputDeviceId(current.selectedDeviceId, inputDevices),
        devicesLoading: false,
        devicesError: null,
      }));
    } catch (error) {
      setState((current) => ({
        ...current,
        devicesLoading: false,
        devicesError: error instanceof Error ? error.message : "Failed to list microphone devices.",
      }));
    }
  }, []);

  const setSelectedDeviceId = useCallback((deviceId: string) => {
    setState((current) => ({
      ...current,
      selectedDeviceId: deviceId,
    }));
  }, []);

  const reset = useCallback(() => {
    mediaRecorderRef.current = null;
    sessionRef.current = null;
    chunkIndexRef.current = 0;
    chunkStartedAtRef.current = null;
    durationRef.current = null;
    acceptingChunksRef.current = false;
    failedChunksRef.current = [];
    uploadChainRef.current = Promise.resolve();
    stopResolverRef.current = null;
    stopStream();
    setState((current) => ({
      status: getBrowserRecordingSupport().supported ? "idle" : "unsupported",
      session: null,
      error: supportErrorMessage(getBrowserRecordingSupport()),
      elapsedMs: 0,
      selectedMimeType: "",
      appliedSettings: null,
      requestedConstraints: null,
      pendingChunks: 0,
      uploadedChunks: 0,
      failedChunkIndex: null,
      availableDevices: current.availableDevices,
      selectedDeviceId: current.selectedDeviceId,
      devicesLoading: current.devicesLoading,
      devicesError: current.devicesError,
    }));
  }, [stopStream]);

  const enqueueChunkUpload = useCallback((chunk: PendingChunk) => {
    setState((current) => ({
      ...current,
      pendingChunks: current.pendingChunks + 1,
      failedChunkIndex: null,
    }));

    uploadChainRef.current = uploadChainRef.current
      .then(async () => {
        const recording = sessionRef.current;
        if (!recording) return;
        const sha256 = await sha256Blob(chunk.blob);
        await uploadChunkMutation.mutateAsync({
          recordingId: recording.id,
          chunkIndex: chunk.index,
          chunk: chunk.blob,
          mimeType: chunk.mimeType,
          sha256,
          durationMs: chunk.durationMs,
        });
        setState((current) => ({
          ...current,
          pendingChunks: Math.max(0, current.pendingChunks - 1),
          uploadedChunks: current.uploadedChunks + 1,
          failedChunkIndex: null,
        }));
      })
      .catch((error: unknown) => {
        const recorder = mediaRecorderRef.current;
        if (recorder?.state === "recording") {
          recorder.pause();
        }
        acceptingChunksRef.current = false;
        if (durationRef.current) {
          durationRef.current = pauseRecordingDuration(durationRef.current, performance.now());
        }
        failedChunksRef.current = [...failedChunksRef.current, chunk];
        setState((current) => ({
          ...current,
          status: "failed",
          error: error instanceof Error ? error.message : "Failed to upload recording chunk",
          pendingChunks: Math.max(0, current.pendingChunks - 1),
          failedChunkIndex: chunk.index,
        }));
      });
  }, [uploadChunkMutation]);

  const start = useCallback(async (options: BrowserRecorderStartOptions) => {
    const support = getBrowserRecordingSupport();
    if (!support.supported) {
      setState((current) => ({
        ...current,
        status: "unsupported",
        error: supportErrorMessage(support),
      }));
      return;
    }

    try {
      const mimeSelection = selectRecordingMimeType();
      const selectedDeviceId = options.deviceId || state.selectedDeviceId;
      const constraints = buildMicrophoneConstraints(selectedDeviceId || undefined);
      const stream = await navigator.mediaDevices.getUserMedia({ audio: constraints.audio });
      const audioTrack = stream.getAudioTracks()[0];
      const appliedSettings = audioTrack?.getSettings() || null;
      const recorder = createMediaRecorder(stream, mimeSelection);
      const chunkDurationMs = options.chunkDurationMs || defaultChunkDurationMs;
      const session = await createRecordingMutation.mutateAsync({
        title: options.title,
        source_kind: "microphone",
        mime_type: recorder.mimeType || mimeSelection.mimeType || "application/octet-stream",
        codec: mimeSelection.codec,
        chunk_duration_ms: chunkDurationMs,
        auto_transcribe: options.autoTranscribe ?? false,
        profile_id: options.profileId,
        options: {
          language: options.language,
          diarization: options.diarization,
        },
      });

      streamRef.current = stream;
      mediaRecorderRef.current = recorder;
      sessionRef.current = session;
      chunkIndexRef.current = 0;
      failedChunksRef.current = [];
      uploadChainRef.current = Promise.resolve();
      durationRef.current = createRecordingDurationState(performance.now());
      chunkStartedAtRef.current = performance.now();
      acceptingChunksRef.current = true;

      recorder.addEventListener("dataavailable", (event) => {
        if (!acceptingChunksRef.current || !event.data || event.data.size === 0) return;
        const now = performance.now();
        const durationMs = chunkStartedAtRef.current === null ? undefined : Math.max(0, now - chunkStartedAtRef.current);
        chunkStartedAtRef.current = now;
        enqueueChunkUpload({
          index: chunkIndexRef.current,
          blob: event.data,
          mimeType: event.data.type || recorder.mimeType || mimeSelection.mimeType || "application/octet-stream",
          durationMs,
        });
        chunkIndexRef.current += 1;
      });

      recorder.addEventListener("stop", () => {
        stopResolverRef.current?.();
        stopResolverRef.current = null;
      });

      recorder.start(chunkDurationMs);
      setState((current) => ({
        ...current,
        status: "recording",
        session,
        error: null,
        elapsedMs: 0,
        selectedMimeType: recorder.mimeType || mimeSelection.mimeType,
        appliedSettings,
        requestedConstraints: constraints.requested,
        pendingChunks: 0,
        uploadedChunks: 0,
        failedChunkIndex: null,
        selectedDeviceId,
        devicesError: null,
      }));
      void refreshInputDevices();
    } catch (error) {
      stopStream();
      setError(error, "Failed to start recording");
    }
  }, [createRecordingMutation, enqueueChunkUpload, refreshInputDevices, setError, state.selectedDeviceId, stopStream]);

  const pause = useCallback(() => {
    const recorder = mediaRecorderRef.current;
    if (!recorder || recorder.state !== "recording" || !durationRef.current) return;
    recorder.pause();
    durationRef.current = pauseRecordingDuration(durationRef.current, performance.now());
    setState((current) => ({
      ...current,
      status: "paused",
      elapsedMs: durationRef.current ? elapsedRecordingDurationMs(durationRef.current, performance.now()) : current.elapsedMs,
    }));
  }, []);

  const resume = useCallback(() => {
    const recorder = mediaRecorderRef.current;
    if (!recorder || recorder.state !== "paused" || !durationRef.current) return;
    recorder.resume();
    const now = performance.now();
    durationRef.current = resumeRecordingDuration(durationRef.current, now);
    chunkStartedAtRef.current = now;
    acceptingChunksRef.current = true;
    setState((current) => ({
      ...current,
      status: "recording",
      error: null,
    }));
  }, []);

  const stop = useCallback(async () => {
    const recorder = mediaRecorderRef.current;
    const session = sessionRef.current;
    if (!recorder || !session || !durationRef.current) return;

    try {
      setState((current) => ({ ...current, status: "stopping", error: null }));
      durationRef.current = stopRecordingDuration(durationRef.current, performance.now());
      const durationMs = elapsedRecordingDurationMs(durationRef.current, performance.now());

      if (recorder.state !== "inactive") {
        const stopped = new Promise<void>((resolve) => {
          stopResolverRef.current = resolve;
        });
        recorder.requestData();
        recorder.stop();
        await stopped;
      }
      acceptingChunksRef.current = false;

      await uploadChainRef.current;
      if (failedChunksRef.current.length > 0) {
        throw new Error("Some recording chunks failed to upload");
      }

      setState((current) => ({ ...current, status: "finalizing", elapsedMs: durationMs }));
      const stoppedSession = await stopRecordingMutation.mutateAsync({
        recordingId: session.id,
        payload: {
          final_chunk_index: Math.max(0, chunkIndexRef.current - 1),
          duration_ms: durationMs,
        },
      });

      sessionRef.current = stoppedSession;
      stopStream();
      setState((current) => ({
        ...current,
        status: stoppedSession.status === "ready" ? "ready" : "finalizing",
        session: stoppedSession,
        elapsedMs: durationMs,
      }));
    } catch (error) {
      setError(error, "Failed to stop recording");
    }
  }, [setError, stopRecordingMutation, stopStream]);

  const cancel = useCallback(async () => {
    const recorder = mediaRecorderRef.current;
    const session = sessionRef.current;

    try {
      acceptingChunksRef.current = false;
      if (recorder && recorder.state !== "inactive") {
        recorder.stop();
      }
      stopStream();
      if (session) {
        await cancelRecordingMutation.mutateAsync(session.id);
      }
      setState((current) => ({
        ...current,
        status: "canceled",
        error: null,
      }));
    } catch (error) {
      setError(error, "Failed to cancel recording");
    }
  }, [cancelRecordingMutation, setError, stopStream]);

  const retryPendingChunk = useCallback(() => {
    const chunks = failedChunksRef.current;
    if (chunks.length === 0) return;
    failedChunksRef.current = [];
    setState((current) => ({
      ...current,
      status: current.session?.status === "stopping" ? "stopping" : "paused",
      error: null,
      failedChunkIndex: null,
    }));
    chunks.forEach(enqueueChunkUpload);
  }, [enqueueChunkUpload]);

  useEffect(() => {
    if (state.status !== "recording") return;
    const timer = window.setInterval(() => {
      const duration = durationRef.current;
      if (!duration) return;
      setState((current) => ({
        ...current,
        elapsedMs: elapsedRecordingDurationMs(duration, performance.now()),
      }));
    }, 250);

    return () => window.clearInterval(timer);
  }, [state.status]);

  useEffect(() => {
    void refreshInputDevices();
  }, [refreshInputDevices]);

  useEffect(() => {
    if (!navigator.mediaDevices?.addEventListener) return;
    const handleDeviceChange = () => {
      void refreshInputDevices();
    };

    navigator.mediaDevices.addEventListener("devicechange", handleDeviceChange);
    return () => navigator.mediaDevices.removeEventListener("devicechange", handleDeviceChange);
  }, [refreshInputDevices]);

  useEffect(() => {
    const shouldWarn = state.status === "recording" || state.status === "paused" || state.pendingChunks > 0;
    if (!shouldWarn) return;

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => window.removeEventListener("beforeunload", handleBeforeUnload);
  }, [state.pendingChunks, state.status]);

  useEffect(() => {
    return () => {
      acceptingChunksRef.current = false;
      const recorder = mediaRecorderRef.current;
      if (recorder && recorder.state !== "inactive") {
        recorder.stop();
      }
      mediaRecorderRef.current = null;
      stopStream();
    };
  }, [stopStream]);

  return {
    state,
    start,
    pause,
    resume,
    stop,
    cancel,
    retryPendingChunk,
    reset,
    refreshInputDevices,
    setSelectedDeviceId,
  };
}

function toRecordingInputDevices(devices: MediaDeviceInfo[]): RecordingInputDevice[] {
  return devices
    .filter((device) => device.kind === "audioinput")
    .map((device, index) => ({
      deviceId: device.deviceId,
      label: device.label || `Microphone ${index + 1}`,
      groupId: device.groupId || undefined,
    }));
}

function selectedInputDeviceId(currentDeviceId: string, devices: RecordingInputDevice[]) {
  if (devices.length === 0) return "";
  if (currentDeviceId && devices.some((device) => device.deviceId === currentDeviceId)) {
    return currentDeviceId;
  }
  return devices[0]?.deviceId || "";
}

function createMediaRecorder(stream: MediaStream, selection: RecordingMimeSelection) {
  const options = mediaRecorderOptionsFor(selection);
  try {
    return options ? new MediaRecorder(stream, options) : new MediaRecorder(stream);
  } catch {
    return new MediaRecorder(stream);
  }
}

async function sha256Blob(blob: Blob): Promise<string | undefined> {
  if (!crypto.subtle) return undefined;
  const buffer = await blob.arrayBuffer();
  const hash = await crypto.subtle.digest("SHA-256", buffer);
  return Array.from(new Uint8Array(hash))
    .map((byte) => byte.toString(16).padStart(2, "0"))
    .join("");
}

function isPermissionError(error: unknown) {
  return error instanceof DOMException && (error.name === "NotAllowedError" || error.name === "SecurityError");
}

function supportErrorMessage(support: BrowserRecordingSupport) {
  switch (support.reason) {
    case "missing-media-devices":
    case "missing-get-user-media":
      return "This browser does not support microphone recording.";
    case "missing-media-recorder":
      return "This browser does not support MediaRecorder.";
    default:
      return null;
  }
}
