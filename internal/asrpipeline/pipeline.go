package asrpipeline

import "scriberr/internal/models"

// DefaultTranscription returns the smallest valid pipeline. Provider/model
// defaults are resolved later by the ASR provider registry from descriptors.
func DefaultTranscription() models.ASRParams {
	return models.ASRParams{
		Pipeline: []models.ASRStep{{Kind: models.ASRStepTranscription}},
	}
}

func EnsureTranscription(params models.ASRParams) models.ASRParams {
	if hasStep(params.Pipeline, models.ASRStepTranscription) {
		return params
	}
	params.Pipeline = append([]models.ASRStep{{Kind: models.ASRStepTranscription}}, params.Pipeline...)
	return params
}

func SetDiarization(params *models.ASRParams, enabled bool) {
	if params == nil {
		return
	}
	filtered := make([]models.ASRStep, 0, len(params.Pipeline)+1)
	var existing *models.ASRStep
	for _, step := range params.Pipeline {
		if step.Kind != models.ASRStepDiarization {
			filtered = append(filtered, step)
			continue
		}
		copied := step
		existing = &copied
	}
	if enabled {
		step := models.ASRStep{Kind: models.ASRStepDiarization}
		if existing != nil {
			step = *existing
		}
		filtered = append(filtered, step)
	}
	params.Pipeline = filtered
}

func HasStep(steps []models.ASRStep, kind string) bool {
	return hasStep(steps, kind)
}

func hasStep(steps []models.ASRStep, kind string) bool {
	for _, step := range steps {
		if step.Kind == kind {
			return true
		}
	}
	return false
}
