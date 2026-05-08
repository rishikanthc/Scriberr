import * as assert from "node:assert/strict";
import { listASRModels, listTranscriptionModels, normalizeProfileOptions, type ASRModelCard } from "./profilesApi";

const transcriptionModel: ASRModelCard = {
  id: "parakeet-v3",
  display_name: "NVIDIA Parakeet TDT v3",
  provider: "local",
  installed: true,
  default: true,
  capabilities: { transcription: true, word_timestamps: true },
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
};

const diarizationModel: ASRModelCard = {
  id: "diarization-default",
  display_name: "Diarization",
  provider: "local",
  installed: true,
  default: false,
  capabilities: { diarization: true },
  recommended_defaults: { "diarization.threshold": 0.5 },
  parameter_schema: [
    { key: "diarization.threshold", type: "number", scope: "model", default: 0.5 },
  ],
};

const originalFetch = globalThis.fetch;
const requestedURLs: string[] = [];

async function main() {
  globalThis.fetch = async (input: RequestInfo | URL) => {
    const url = String(input);
    requestedURLs.push(url);
    return new Response(JSON.stringify({ items: [transcriptionModel, diarizationModel] }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    });
  };

  const transcriptionModels = await listTranscriptionModels({ Authorization: "Bearer test" });
  assert.equal(requestedURLs[0], "/api/v1/models?capability=transcription");
  assert.deepEqual(transcriptionModels, [transcriptionModel]);
  assert.equal(transcriptionModels[0].parameter_schema?.[2].read_only, true);
  assert.equal(transcriptionModels[0].recommended_defaults?.["runtime.num_threads"], 4);

  const diarizationModels = await listASRModels(["diarization"]);
  assert.equal(requestedURLs[1], "/api/v1/models?capability=diarization");
  assert.deepEqual(diarizationModels, [diarizationModel]);

  assert.deepEqual(normalizeProfileOptions(undefined), { pipeline: [] });
}

main()
  .finally(() => {
    globalThis.fetch = originalFetch;
  })
  .catch((error) => {
    throw error;
  });
