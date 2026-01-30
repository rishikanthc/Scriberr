package adapters

import (
	"fmt"
	"strconv"
)

func buildEngineParams(params map[string]interface{}) map[string]string {
	engineParams := map[string]string{}

	if timestamps, ok := params["timestamps"].(bool); ok && !timestamps {
		engineParams["include_segments"] = "false"
		engineParams["include_words"] = "false"
	} else {
		engineParams["include_segments"] = "true"
		engineParams["include_words"] = "true"
	}

	if diarize, ok := params["diarize"].(bool); ok && diarize {
		if _, has := params["vad_enabled"]; !has {
			engineParams["vad_enabled"] = "false"
		}
	}

	if val, ok := params["vad_enabled"].(bool); ok {
		if val {
			engineParams["vad_enabled"] = "true"
		} else {
			engineParams["vad_enabled"] = "false"
		}
	}

	if val, ok := params["vad_preset"].(string); ok && val != "" {
		engineParams["vad_preset"] = val
	}
	if val, ok := params["vad_speech_pad_ms"]; ok {
		if intVal, err := toInt(val); err == nil {
			engineParams["vad_speech_pad_ms"] = strconv.Itoa(intVal)
		}
	}
	if val, ok := params["vad_min_silence_ms"]; ok {
		if intVal, err := toInt(val); err == nil {
			engineParams["vad_min_silence_ms"] = strconv.Itoa(intVal)
		}
	}
	if val, ok := params["vad_min_speech_ms"]; ok {
		if intVal, err := toInt(val); err == nil {
			engineParams["vad_min_speech_ms"] = strconv.Itoa(intVal)
		}
	}
	if val, ok := params["vad_max_speech_s"]; ok {
		if intVal, err := toInt(val); err == nil {
			engineParams["vad_max_speech_s"] = strconv.Itoa(intVal)
		}
	}

	if lang, ok := params["language"].(string); ok && lang != "" {
		engineParams["language"] = lang
	}
	if target, ok := params["target_language"].(string); ok && target != "" {
		engineParams["target_language"] = target
	}
	if pncVal, ok := params["pnc"].(bool); ok {
		if pncVal {
			engineParams["pnc"] = "true"
		} else {
			engineParams["pnc"] = "false"
		}
	} else if pncStr, ok := params["pnc"].(string); ok && pncStr != "" {
		engineParams["pnc"] = pncStr
	}

	return engineParams
}

func toInt(val interface{}) (int, error) {
	switch v := val.(type) {
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case float32:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("unsupported int type %T", val)
	}
}
