export type RecordingDurationState = {
  accumulatedMs: number;
  activeStartedAt: number | null;
};

export function createRecordingDurationState(now: number): RecordingDurationState {
  return {
    accumulatedMs: 0,
    activeStartedAt: now,
  };
}

export function pauseRecordingDuration(state: RecordingDurationState, now: number): RecordingDurationState {
  if (state.activeStartedAt === null) return state;
  return {
    accumulatedMs: state.accumulatedMs + Math.max(0, now - state.activeStartedAt),
    activeStartedAt: null,
  };
}

export function resumeRecordingDuration(state: RecordingDurationState, now: number): RecordingDurationState {
  if (state.activeStartedAt !== null) return state;
  return {
    ...state,
    activeStartedAt: now,
  };
}

export function stopRecordingDuration(state: RecordingDurationState, now: number): RecordingDurationState {
  return pauseRecordingDuration(state, now);
}

export function elapsedRecordingDurationMs(state: RecordingDurationState, now: number): number {
  if (state.activeStartedAt === null) return Math.round(state.accumulatedMs);
  return Math.round(state.accumulatedMs + Math.max(0, now - state.activeStartedAt));
}
