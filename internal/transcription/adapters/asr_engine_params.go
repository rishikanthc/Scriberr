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
		_ = diarize
	}

	if val, ok := params["chunk_len_s"]; ok {
		if floatVal, err := toFloat(val); err == nil {
			engineParams["chunk_len_s"] = strconv.FormatFloat(floatVal, 'f', -1, 64)
		}
	}
	if val, ok := params["chunk_batch_size"]; ok {
		if intVal, err := toInt(val); err == nil {
			engineParams["chunk_batch_size"] = strconv.Itoa(intVal)
		}
	}
	if val, ok := params["segment_gap_s"]; ok {
		if floatVal, err := toFloat(val); err == nil {
			engineParams["segment_gap_s"] = strconv.FormatFloat(floatVal, 'f', -1, 64)
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

func toFloat(val interface{}) (float64, error) {
	switch v := val.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported float type %T", val)
	}
}
