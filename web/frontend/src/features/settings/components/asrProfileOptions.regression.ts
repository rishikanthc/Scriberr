import * as assert from "node:assert/strict";
import type { TranscriptionModel, TranscriptionProfileOptions } from "../api/profilesApi";
import { prepareProfileOptionsForSave, withDiarizationModel, withTranscriptionModel } from "./asrProfileOptions";

const transcriptionModels: TranscriptionModel[] = [
  {
    id: "parakeet-v3",
    display_name: "NVIDIA Parakeet TDT v3",
    provider: "local",
    installed: true,
    default: true,
    capabilities: { transcription: true },
    recommended_defaults: {
      "chunking.mode": "fixed",
      "runtime.num_threads": 4,
      "sherpa.model_type": "nemo_transducer",
    },
    parameter_schema: [
      { key: "chunking.mode", type: "enum", scope: "chunking", default: "fixed", options: [{ value: "fixed" }, { value: "vad" }] },
      { key: "runtime.num_threads", type: "integer", scope: "runtime", default: 1 },
      { key: "sherpa.model_type", type: "string", scope: "model", default: "nemo_transducer", read_only: true },
    ],
  },
];

const diarizationModels: TranscriptionModel[] = [
  {
    id: "diarization-default",
    display_name: "Diarization",
    provider: "local",
    installed: true,
    default: true,
    capabilities: { diarization: true },
    recommended_defaults: { "diarization.threshold": 0.5 },
    parameter_schema: [
      { key: "diarization.threshold", type: "number", scope: "model", default: 0.5 },
      { key: "diarization.num_clusters", type: "integer", scope: "model", default: 0 },
    ],
  },
];

let options: TranscriptionProfileOptions = { pipeline: [] };
options = withTranscriptionModel(options, transcriptionModels, "parakeet-v3");
assert.deepEqual(options.pipeline[0], {
  kind: "transcription",
  provider: "local",
  model: "parakeet-v3",
  options: {
    "chunking.mode": "fixed",
    "runtime.num_threads": 4,
    "sherpa.model_type": "nemo_transducer",
  },
});

options = {
  pipeline: [{
    ...options.pipeline[0],
    options: {
      "chunking.mode": "vad",
      "runtime.num_threads": 8,
      unsupported: true,
    },
  }],
};
options = withDiarizationModel(options, diarizationModels, "diarization-default");

const saveOptions = prepareProfileOptionsForSave(options, transcriptionModels, diarizationModels);
assert.deepEqual(saveOptions.pipeline.map((step) => step.kind), ["transcription", "diarization"]);
assert.deepEqual(saveOptions.pipeline[0].options, {
  "chunking.mode": "vad",
  "runtime.num_threads": 8,
  "sherpa.model_type": "nemo_transducer",
});
assert.deepEqual(saveOptions.pipeline[1].options, {
  "diarization.threshold": 0.5,
  "diarization.num_clusters": 0,
});
