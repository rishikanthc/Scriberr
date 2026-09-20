package transcription

import (
	"scriberr/internal/models"
	"testing"
)

func TestOrukeetModelRouting(t *testing.T) {
	service := &UnifiedTranscriptionService{}
	for _, test := range []struct{ family, model, want string }{
		{FamilyNvidiaParakeet, "orukeet-v0.1.0", ModelOrukeet},
		{FamilyNvidiaParakeet, "parakeet-tdt-0.6b-v3", ModelParakeet},
		{FamilyNvidiaParakeet, "small", ModelParakeet},
		{FamilyWhisper, "orukeet-v0.1.0", ModelWhisperX},
	} {
		got, diarization, err := service.selectModels(models.WhisperXParams{ModelFamily: test.family, Model: test.model, Diarize: true, DiarizeModel: ModelPyannote})
		if err != nil || got != test.want || diarization != ModelPyannote {
			t.Fatalf("%+v: %s %s %v", test, got, diarization, err)
		}
	}
}
