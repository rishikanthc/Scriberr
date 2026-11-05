package adapters

import (
	"slices"
	"testing"

	"scriberr/internal/transcription/interfaces"
)

func TestBuildWhisperXArgsPassesInitialPrompt(t *testing.T) {
	adapter := NewWhisperXAdapter(t.TempDir())
	input := interfaces.AudioInput{FilePath: "audio.wav"}

	args, err := adapter.buildWhisperXArgs(input, map[string]interface{}{"initial_prompt": "WhisperX, PyAnnote"}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	i := slices.Index(args, "--initial_prompt")
	if i < 0 || args[i+1] != "WhisperX, PyAnnote" {
		t.Fatalf("initial prompt not passed: %v", args)
	}

	args, err = adapter.buildWhisperXArgs(input, map[string]interface{}{}, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if slices.Contains(args, "--initial_prompt") {
		t.Fatalf("empty prompt passed: %v", args)
	}
}
