package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"scriberr/internal/transcription/interfaces"
)

// Opt in with SCRIBERR_NEMO_TEST_ENV (the prepared NeMo project directory)
// and SCRIBERR_NEMO_TEST_AUDIO (a spoken 16 kHz mono WAV longer than 5 s).
// This restores real checkpoints and invokes the same subprocesses as jobs.
func TestOrukeetRealInference(t *testing.T) {
	env, audio := os.Getenv("SCRIBERR_NEMO_TEST_ENV"), os.Getenv("SCRIBERR_NEMO_TEST_AUDIO")
	if env == "" || audio == "" {
		t.Skip("set SCRIBERR_NEMO_TEST_ENV and SCRIBERR_NEMO_TEST_AUDIO for real inference")
	}
	stat, err := os.Stat(audio)
	if err != nil {
		t.Fatal(err)
	}
	input := interfaces.AudioInput{FilePath: audio, Format: "wav", SampleRate: 16000, Channels: 1, Size: stat.Size()}
	adapter := NewOrukeetAdapter(env)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	if err := adapter.PrepareEnvironment(ctx); err != nil {
		t.Fatal(err)
	}
	params := map[string]interface{}{"timestamps": true, "context_left": 256, "context_right": 256, "auto_convert_audio": false}
	run := func(id string) (*interfaces.TranscriptResult, error) {
		dir := filepath.Join(t.TempDir(), id)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		result, err := adapter.Transcribe(ctx, input, params, interfaces.ProcessingContext{JobID: id, TempDirectory: dir, OutputDirectory: dir})
		if err != nil {
			return nil, err
		}
		if result.ModelUsed != "orukeet-v0.1.0" || result.Language != "und" || strings.TrimSpace(result.Text) == "" || len(result.WordSegments) == 0 {
			return nil, fmt.Errorf("invalid transcript: %+v", result)
		}
		for _, word := range result.WordSegments {
			if word.Start < 0 || word.End < word.Start {
				return nil, fmt.Errorf("invalid word times: %+v", word)
			}
		}
		encoded, _ := json.Marshal(result)
		t.Logf("%s: %s", id, encoded)
		return result, nil
	}
	if _, err := run("standard"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PARAKEET_CHUNK_THRESHOLD_SECS", "5")
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, err := run(fmt.Sprintf("buffered-%d", i)); errors <- err }(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	cancelled, stop := context.WithCancel(ctx)
	stop()
	if _, err := adapter.Transcribe(cancelled, input, params, interfaces.ProcessingContext{}); err == nil {
		t.Fatal("cancelled request succeeded")
	}
}
