export type RecordingMimeSelection = {
  mimeType: string;
  codec?: string;
};

export type MicrophoneConstraintSelection = {
  audio: MediaTrackConstraints;
  requested: {
    echoCancellation: boolean;
    noiseSuppression: boolean;
    autoGainControl: boolean;
    channelCount: boolean;
    sampleRate: boolean;
    deviceId: boolean;
  };
};

export type BrowserRecordingSupport = {
  supported: boolean;
  reason?: "missing-media-devices" | "missing-get-user-media" | "missing-media-recorder";
};

const preferredAudioMimeTypes = [
  "audio/webm;codecs=opus",
  "audio/webm",
  "audio/ogg;codecs=opus",
  "audio/ogg",
  "audio/mp4;codecs=mp4a.40.2",
  "audio/mp4",
  "audio/aac",
] as const;

export function getBrowserRecordingSupport(): BrowserRecordingSupport {
  if (!navigator.mediaDevices) {
    return { supported: false, reason: "missing-media-devices" };
  }
  if (!navigator.mediaDevices.getUserMedia) {
    return { supported: false, reason: "missing-get-user-media" };
  }
  if (typeof MediaRecorder === "undefined") {
    return { supported: false, reason: "missing-media-recorder" };
  }
  return { supported: true };
}

export function selectRecordingMimeType(): RecordingMimeSelection {
  if (typeof MediaRecorder === "undefined" || typeof MediaRecorder.isTypeSupported !== "function") {
    return { mimeType: "" };
  }

  const mimeType = preferredAudioMimeTypes.find((candidate) => MediaRecorder.isTypeSupported(candidate)) || "";
  return {
    mimeType,
    codec: codecFromMimeType(mimeType),
  };
}

export function buildMicrophoneConstraints(deviceId?: string): MicrophoneConstraintSelection {
  const supported = navigator.mediaDevices?.getSupportedConstraints?.() || {};
  const audio: MediaTrackConstraints = {};

  if (deviceId) {
    audio.deviceId = { exact: deviceId };
  }
  if (supported.echoCancellation) {
    audio.echoCancellation = true;
  }
  if (supported.noiseSuppression) {
    audio.noiseSuppression = true;
  }
  if (supported.autoGainControl) {
    audio.autoGainControl = true;
  }
  if (supported.channelCount) {
    audio.channelCount = { ideal: 1 };
  }
  if (supported.sampleRate) {
    audio.sampleRate = { ideal: 48000 };
  }

  return {
    audio,
    requested: {
      echoCancellation: Boolean(supported.echoCancellation),
      noiseSuppression: Boolean(supported.noiseSuppression),
      autoGainControl: Boolean(supported.autoGainControl),
      channelCount: Boolean(supported.channelCount),
      sampleRate: Boolean(supported.sampleRate),
      deviceId: Boolean(deviceId),
    },
  };
}

export function mediaRecorderOptionsFor(selection: RecordingMimeSelection): MediaRecorderOptions | undefined {
  if (!selection.mimeType) return undefined;
  return {
    mimeType: selection.mimeType,
    audioBitsPerSecond: 128_000,
  };
}

function codecFromMimeType(mimeType: string) {
  const codecMatch = mimeType.match(/codecs=([^;]+)/i);
  return codecMatch?.[1]?.trim();
}
