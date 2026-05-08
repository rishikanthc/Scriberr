import { useMemo } from "react";
import { Check } from "lucide-react";
import { Select, type SelectOption } from "@/shared/ui/Select";
import type { ASRModelCard, ParameterDescriptor } from "../api/profilesApi";
import { groupByScope, resolveParameterValues, sanitizeParameterValues, visibleParameterSchema } from "./asrParameterValues";

type ASRParameterFormProps = {
  model: ASRModelCard | null;
  values: Record<string, unknown>;
  onChange: (values: Record<string, unknown>) => void;
};

const scopeLabels: Record<ParameterDescriptor["scope"], string> = {
  model: "Model",
  runtime: "Runtime",
  decoding: "Decoding",
  chunking: "Chunking",
  vad: "Voice activity",
  output: "Output",
  postprocess: "Postprocess",
};

export function ASRParameterForm({ model, values, onChange }: ASRParameterFormProps) {
  const resolvedValues = useMemo(() => resolveParameterValues(model, values), [model, values]);
  const visibleParameters = useMemo(() => visibleParameterSchema(model, resolvedValues), [model, resolvedValues]);
  const standardParameters = visibleParameters.filter((parameter) => !parameter.advanced);
  const advancedParameters = visibleParameters.filter((parameter) => parameter.advanced);

  if (!model) {
    return <div className="scr-alert">Select a model to configure parameters.</div>;
  }

  if (visibleParameters.length === 0) {
    return null;
  }

  const updateValue = (key: string, value: unknown) => {
    onChange(sanitizeParameterValues(model, { ...resolvedValues, [key]: value }));
  };

  return (
    <div className="scr-asr-parameter-form">
      <ParameterGroups parameters={standardParameters} values={resolvedValues} onChange={updateValue} />
      {advancedParameters.length > 0 ? (
        <details className="scr-asr-advanced">
          <summary>Advanced</summary>
          <ParameterGroups parameters={advancedParameters} values={resolvedValues} onChange={updateValue} />
        </details>
      ) : null}
    </div>
  );
}

function ParameterGroups({
  parameters,
  values,
  onChange,
}: {
  parameters: ParameterDescriptor[];
  values: Record<string, unknown>;
  onChange: (key: string, value: unknown) => void;
}) {
  const groups = groupByScope(parameters);
  return (
    <>
      {groups.map(([scope, scopeParameters]) => (
        <section className="scr-settings-section" key={scope}>
          <h3 className="scr-settings-section-title">{scopeLabels[scope]}</h3>
          <div className="scr-form-grid">
            {scopeParameters.map((parameter) => (
              <ParameterControl
                key={parameter.key}
                parameter={parameter}
                value={values[parameter.key]}
                onChange={(value) => onChange(parameter.key, value)}
              />
            ))}
          </div>
        </section>
      ))}
    </>
  );
}

function ParameterControl({ parameter, value, onChange }: { parameter: ParameterDescriptor; value: unknown; onChange: (value: unknown) => void }) {
  const label = parameter.label || parameter.key;
  const disabled = parameter.read_only === true;

  if (parameter.type === "boolean") {
    return (
      <label className="scr-check-row">
        <input type="checkbox" checked={Boolean(value)} disabled={disabled} onChange={(event) => onChange(event.target.checked)} />
        <span className="scr-check-box" aria-hidden="true">{value ? <Check size={13} /> : null}</span>
        <span>{label}</span>
      </label>
    );
  }

  if (parameter.type === "enum") {
    const options: SelectOption[] = (parameter.options || []).map((option) => ({
      value: String(option.value),
      label: option.label || String(option.value),
    }));
    return <Select label={label} value={value == null ? "" : String(value)} options={options} disabled={disabled} onChange={(nextValue) => onChange(parseEnumValue(parameter, nextValue))} />;
  }

  if (parameter.type === "integer" || parameter.type === "number" || parameter.type === "duration") {
    return (
      <label className="scr-control">
        <span>{label}</span>
        <input
          className="scr-input"
          type="number"
          min={parameter.min}
          max={parameter.max}
          step={parameter.step || (parameter.type === "integer" ? 1 : "any")}
          value={value == null ? "" : String(value)}
          disabled={disabled}
          onChange={(event) => onChange(parseNumberValue(parameter, event.target.value))}
        />
      </label>
    );
  }

  return (
    <label className="scr-control">
      <span>{label}</span>
      <input className="scr-input" value={value == null ? "" : String(value)} disabled={disabled} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function parseEnumValue(parameter: ParameterDescriptor, value: string) {
  const option = (parameter.options || []).find((item) => String(item.value) === value);
  return option ? option.value : value;
}

function parseNumberValue(parameter: ParameterDescriptor, value: string) {
  if (value === "") return undefined;
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return undefined;
  return parameter.type === "integer" ? Math.trunc(parsed) : parsed;
}
