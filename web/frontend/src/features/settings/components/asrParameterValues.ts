import type { ASRModelCard, ParameterDescriptor } from "../api/profilesApi";

export function resolveParameterValues(model: ASRModelCard | null | undefined, values: Record<string, unknown>) {
  const resolved: Record<string, unknown> = {};
  for (const parameter of model?.parameter_schema || []) {
    if (Object.prototype.hasOwnProperty.call(values, parameter.key)) {
      resolved[parameter.key] = values[parameter.key];
    } else if (model?.recommended_defaults && Object.prototype.hasOwnProperty.call(model.recommended_defaults, parameter.key)) {
      resolved[parameter.key] = model.recommended_defaults[parameter.key];
    } else if (Object.prototype.hasOwnProperty.call(parameter, "default")) {
      resolved[parameter.key] = parameter.default;
    }
  }
  return resolved;
}

export function sanitizeParameterValues(model: ASRModelCard | null | undefined, values: Record<string, unknown>) {
  const allowed = new Set((model?.parameter_schema || []).map((parameter) => parameter.key));
  const out: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(values)) {
    if (allowed.has(key) && value !== undefined && value !== "") {
      out[key] = value;
    }
  }
  return out;
}

export function visibleParameterSchema(model: ASRModelCard | null | undefined, values: Record<string, unknown>) {
  return (model?.parameter_schema || []).filter((parameter) => {
    return (parameter.visible_when || []).every((rule) => {
      if (rule.operator !== "equals") return false;
      return values[rule.parameter] === rule.value;
    });
  });
}

export function groupByScope(parameters: ParameterDescriptor[]) {
  const groups = new Map<ParameterDescriptor["scope"], ParameterDescriptor[]>();
  for (const parameter of parameters) {
    groups.set(parameter.scope, [...(groups.get(parameter.scope) || []), parameter]);
  }
  return Array.from(groups.entries());
}
