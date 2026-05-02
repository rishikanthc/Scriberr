package recording

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultDirPerm os.FileMode = 0o755
const defaultFilePerm os.FileMode = 0o600

type Storage struct {
	root string
}

func NewStorage(root string) (*Storage, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("recording storage root is required")
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve recording storage root: %w", err)
	}
	return &Storage{root: filepath.Clean(absRoot)}, nil
}

func (s *Storage) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *Storage) SessionDir(sessionID string) (string, error) {
	if err := validateStorageID("recording session id", sessionID); err != nil {
		return "", err
	}
	return s.safeJoin(sessionID)
}

func (s *Storage) ChunkDir(sessionID string) (string, error) {
	if err := validateStorageID("recording session id", sessionID); err != nil {
		return "", err
	}
	return s.safeJoin(sessionID, "chunks")
}

func (s *Storage) ChunkPath(sessionID string, chunkIndex int, mimeType string) (string, error) {
	if chunkIndex < 0 {
		return "", fmt.Errorf("recording chunk index cannot be negative")
	}
	chunkDir, err := s.ChunkDir(sessionID)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("%06d%s", chunkIndex, extensionForMimeType(mimeType))
	return s.ensureWithinRoot(filepath.Join(chunkDir, name))
}

func (s *Storage) RawPath(sessionID string) (string, error) {
	if err := validateStorageID("recording session id", sessionID); err != nil {
		return "", err
	}
	return s.safeJoin(sessionID, "raw.webm")
}

func (s *Storage) FinalPath(sessionID string, mimeType string) (string, error) {
	if err := validateStorageID("recording session id", sessionID); err != nil {
		return "", err
	}
	return s.safeJoin(sessionID, "final"+extensionForMimeType(mimeType))
}

func (s *Storage) WriteChunk(ctx context.Context, sessionID string, chunkIndex int, mimeType string, source io.Reader) (string, int64, error) {
	if source == nil {
		return "", 0, fmt.Errorf("recording chunk source is required")
	}
	if err := ctx.Err(); err != nil {
		return "", 0, err
	}
	path, err := s.ChunkPath(sessionID, chunkIndex, mimeType)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(path), defaultDirPerm); err != nil {
		return "", 0, fmt.Errorf("prepare recording chunk directory: %w", err)
	}
	tmpPath := filepath.Join(filepath.Dir(path), "."+filepath.Base(path)+"."+randomSuffix()+".tmp")
	tmpPath, err = s.ensureWithinRoot(tmpPath)
	if err != nil {
		return "", 0, err
	}
	file, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, defaultFilePerm)
	if err != nil {
		return "", 0, fmt.Errorf("create recording chunk temp file: %w", err)
	}
	written, copyErr := copyWithContext(ctx, file, source)
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("write recording chunk: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("close recording chunk: %w", closeErr)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return "", 0, fmt.Errorf("commit recording chunk: %w", err)
	}
	return path, written, nil
}

func (s *Storage) RemoveChunks(sessionID string) error {
	chunkDir, err := s.ChunkDir(sessionID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(chunkDir); err != nil {
		return fmt.Errorf("remove recording chunks: %w", err)
	}
	return nil
}

func (s *Storage) RemoveTemporaryArtifacts(sessionID string) error {
	if err := s.RemoveChunks(sessionID); err != nil {
		return err
	}
	rawPath, err := s.RawPath(sessionID)
	if err != nil {
		return err
	}
	if err := removeIfExists(rawPath); err != nil {
		return fmt.Errorf("remove recording raw artifact: %w", err)
	}
	return nil
}

func (s *Storage) RemoveSession(sessionID string) error {
	sessionDir, err := s.SessionDir(sessionID)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("remove recording session directory: %w", err)
	}
	return nil
}

func (s *Storage) safeJoin(parts ...string) (string, error) {
	if s == nil || s.root == "" {
		return "", fmt.Errorf("recording storage is not configured")
	}
	all := append([]string{s.root}, parts...)
	return s.ensureWithinRoot(filepath.Join(all...))
}

func (s *Storage) ensureWithinRoot(path string) (string, error) {
	if s == nil || s.root == "" {
		return "", fmt.Errorf("recording storage is not configured")
	}
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(s.root, cleanPath)
	if err != nil {
		return "", fmt.Errorf("validate recording storage path: %w", err)
	}
	if rel == "." || rel == "" {
		return cleanPath, nil
	}
	if strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." || filepath.IsAbs(rel) {
		return "", fmt.Errorf("recording storage path escapes root")
	}
	return cleanPath, nil
}

func validateStorageID(label string, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", label)
	}
	if strings.ContainsAny(value, `/\`+"\x00") || value == "." || value == ".." {
		return fmt.Errorf("%s is invalid", label)
	}
	return nil
}

func extensionForMimeType(mimeType string) string {
	base, _, err := mime.ParseMediaType(strings.TrimSpace(mimeType))
	if err != nil {
		base = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	}
	switch base {
	case "audio/webm", "video/webm":
		return ".webm"
	case "audio/ogg":
		return ".ogg"
	case "audio/wav", "audio/wave", "audio/x-wav":
		return ".wav"
	case "audio/flac":
		return ".flac"
	case "audio/mpeg":
		return ".mp3"
	default:
		return ".bin"
	}
}

func randomSuffix() string {
	var bytes [8]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return strconv.FormatInt(int64(os.Getpid()), 10)
	}
	return hex.EncodeToString(bytes[:])
}

func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				return written, ew
			}
			if nr != nw {
				return written, io.ErrShortWrite
			}
		}
		if er != nil {
			if er == io.EOF {
				return written, nil
			}
			return written, er
		}
	}
}

func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
