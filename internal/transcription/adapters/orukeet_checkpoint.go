package adapters

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"scriberr/pkg/downloader"
)

type nemoCheckpoint struct {
	ID, Name, Filename, URL, SHA256 string
	Bytes                           int64
}

var parakeetCheckpoint = nemoCheckpoint{
	ID: "parakeet", Name: "parakeet-tdt-0.6b-v3", Filename: "parakeet-tdt-0.6b-v3.nemo",
}

var orukeetCheckpoint = nemoCheckpoint{
	ID: "orukeet", Name: "orukeet-v0.1.0", Filename: "orukeet-v0.1.0.nemo",
	URL:    "https://huggingface.co/oruk/orukeet/resolve/555136b50265a132d4cea0d35560c26fc4f657ab/orukeet-v0.1.0.nemo",
	SHA256: "031c8ddab4845aeced904a7cde8e8aa57993b2e344716cf83a545b079c473b56", Bytes: 2509342720,
}

func verifyNemoCheckpoint(ctx context.Context, filename string, model nemoCheckpoint) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	if !stat.Mode().IsRegular() || stat.Size() != model.Bytes {
		return fmt.Errorf("checkpoint size mismatch")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, checkpointReader{ctx: ctx, reader: file}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != model.SHA256 {
		return fmt.Errorf("checkpoint checksum mismatch")
	}
	return nil
}

func downloadNemoCheckpoint(ctx context.Context, directory string, model nemoCheckpoint) error {
	destination := filepath.Join(directory, model.Filename)
	if verifyNemoCheckpoint(ctx, destination, model) == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	staged, err := os.CreateTemp(directory, ".orukeet-download-*")
	if err != nil {
		return err
	}
	staging := staged.Name()
	defer os.Remove(staging)
	if err := staged.Close(); err != nil {
		return err
	}
	defer os.Remove(staging + ".tmp")
	downloadCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	if err := downloader.DownloadFile(downloadCtx, model.URL, staging); err != nil {
		return err
	}
	if err := verifyNemoCheckpoint(ctx, staging, model); err != nil {
		return err
	}
	return os.Rename(staging, destination)
}

type checkpointReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r checkpointReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(buffer)
}

func (p *ParakeetAdapter) prepareOrukeet(ctx context.Context) error {
	select {
	case p.prepareLock <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-p.prepareLock }()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !p.orukeetEnvironmentReady {
		if err := p.copyTranscriptionScript(); err != nil {
			return err
		}
		if err := p.copyBufferedScript(); err != nil {
			return err
		}
		check := exec.CommandContext(ctx, "uv", "run", "--native-tls", "--project", p.envPath, "python", "-c", "import nemo.collections.asr")
		if check.Run() != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := p.setupParakeetEnvironment(ctx); err != nil {
				return err
			}
		}
		p.orukeetEnvironmentReady = true
	}
	return downloadNemoCheckpoint(ctx, p.envPath, p.model)
}
