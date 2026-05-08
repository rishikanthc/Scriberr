import * as assert from "node:assert/strict";
import type { ASRModelCard } from "../api/profilesApi";
import { resolveParameterValues, sanitizeParameterValues, visibleParameterSchema } from "./asrParameterValues";

const model: ASRModelCard = {
  id: "parakeet-v3",
  display_name: "NVIDIA Parakeet TDT v3",
  provider: "local",
  installed: true,
  default: false,
  capabilities: { transcription: true },
  recommended_defaults: {
    "chunking.mode": "fixed",
    "runtime.num_threads": 4,
  },
  parameter_schema: [
    {
      key: "chunking.mode",
      label: "Chunking",
      type: "enum",
      scope: "chunking",
      default: "fixed",
      options: [
        { value: "fixed", label: "Fixed" },
        { value: "vad", label: "VAD" },
      ],
      expose_in_summary: true,
    },
    {
      key: "runtime.num_threads",
      label: "Threads",
      type: "integer",
      scope: "runtime",
      default: 1,
      advanced: true,
    },
    {
      key: "vad.threshold",
      label: "VAD threshold",
      type: "number",
      scope: "vad",
      default: 0.5,
      visible_when: [{ parameter: "chunking.mode", operator: "equals", value: "vad" }],
    },
    {
      key: "sherpa.model_type",
      label: "Sherpa model type",
      type: "string",
      scope: "model",
      default: "nemo_transducer",
      read_only: true,
    },
  ],
};

const fixedValues = resolveParameterValues(model, {});
assert.equal(fixedValues["chunking.mode"], "fixed");
assert.equal(fixedValues["runtime.num_threads"], 4);
assert.equal(fixedValues["vad.threshold"], 0.5);
assert.equal(fixedValues["sherpa.model_type"], "nemo_transducer");

assert.deepEqual(
  visibleParameterSchema(model, fixedValues).map((parameter) => parameter.key),
  ["chunking.mode", "runtime.num_threads", "sherpa.model_type"],
);

const vadValues = resolveParameterValues(model, { "chunking.mode": "vad", unsupported: true });
assert.deepEqual(
  visibleParameterSchema(model, vadValues).map((parameter) => parameter.key),
  ["chunking.mode", "runtime.num_threads", "vad.threshold", "sherpa.model_type"],
);

assert.deepEqual(sanitizeParameterValues(model, { "chunking.mode": "vad", unsupported: true, "runtime.num_threads": "" }), {
  "chunking.mode": "vad",
});
