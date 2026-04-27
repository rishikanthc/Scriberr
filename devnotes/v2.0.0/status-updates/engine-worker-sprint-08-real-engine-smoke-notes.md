# EWI-Sprint 8 Real Engine Smoke Notes

Status: opt-in local validation.

Default CI remains fake-provider only. Real engine tests are skipped unless explicitly enabled:

```sh
SCRIBERR_ENGINE_ITEST=1 SPEECH_ENGINE_AUTO_DOWNLOAD=true GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider -run 'TestRealEngine'
```

Primary fixture:

- `test-audio/jfk.wav`: fast transcription smoke fixture.

Optional local timing:

```sh
SCRIBERR_ENGINE_ITEST=1 GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider -bench BenchmarkRealEngineJFKTranscription -run '^$' -benchtime=1x
```

Optional cache override:

```sh
SCRIBERR_ENGINE_ITEST=1 SCRIBERR_ENGINE_ITEST_CACHE_DIR=data/models GOCACHE=/tmp/scriberr-go-cache go test ./internal/transcription/engineprovider -run TestRealEngineJFKTranscription
```

Manual performance notes should record fixture name, cache directory override if any, provider mode, auto-download setting, model download time if observed, transcription wall time, and output text/word counts.

Larger local-only fixtures:

- `test-audio/sample.wav`: broader smoke fixture.
- `test-audio/linus.wav`: longer local validation.
- `test-audio/40min.wav`: performance-only manual validation.
