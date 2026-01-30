package diarengine

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"scriberr/internal/asrengine/pb"
	"scriberr/pkg/logger"

	"github.com/google/shlex"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	defaultSocketPath = "/run/scriberr/engines/diar.sock"
	defaultCmd        = "diar-engine-server"
	defaultTimeoutMs  = 15000
)

type Config struct {
	SocketPath     string
	Command        []string
	StartTimeout   time.Duration
	Providers      []string
	IntraOpThreads int
}

type Manager struct {
	cfg   Config
	mu    sync.Mutex
	cmd   *exec.Cmd
	conn  *grpc.ClientConn
	stub  pb.AsrEngineClient
	jobMu sync.Mutex
}

var (
	defaultOnce    sync.Once
	defaultManager *Manager
)

func Default() *Manager {
	defaultOnce.Do(func() {
		defaultManager = NewManager(LoadConfigFromEnv())
	})
	return defaultManager
}

func NewManager(cfg Config) *Manager {
	return &Manager{cfg: cfg}
}

func LoadConfigFromEnv() Config {
	socketPath := getenv("DIAR_ENGINE_SOCKET", defaultSocketPath)
	cmdStr := strings.TrimSpace(getenv("DIAR_ENGINE_CMD", defaultCmd))
	cmdParts, err := shlex.Split(cmdStr)
	if err != nil || len(cmdParts) == 0 {
		cmdParts = []string{defaultCmd}
	}

	timeoutMs := defaultTimeoutMs
	if val := getenv("DIAR_ENGINE_START_TIMEOUT_MS", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			timeoutMs = parsed
		}
	}

	var providers []string
	if val := getenv("DIAR_ENGINE_PROVIDERS", ""); val != "" {
		for _, p := range strings.Split(val, ",") {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				providers = append(providers, trimmed)
			}
		}
	}

	intraOpThreads := 0
	if val := getenv("DIAR_ENGINE_INTRA_OP_THREADS", ""); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			intraOpThreads = parsed
		}
	}

	return Config{
		SocketPath:     socketPath,
		Command:        cmdParts,
		StartTimeout:   time.Duration(timeoutMs) * time.Millisecond,
		Providers:      providers,
		IntraOpThreads: intraOpThreads,
	}
}

func (m *Manager) EnsureRunning(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.conn != nil {
		if err := m.ping(ctx); err == nil {
			return nil
		}
		m.closeConn()
	}

	if err := m.startProcess(); err != nil {
		return err
	}

	deadline := time.Now().Add(m.cfg.StartTimeout)
	for time.Now().Before(deadline) {
		if err := m.dial(ctx); err == nil {
			if err := m.ping(ctx); err == nil {
				return nil
			}
		}
		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("diarization engine did not become ready within %s", m.cfg.StartTimeout)
}

func (m *Manager) LoadModel(ctx context.Context, spec pb.ModelSpec) error {
	if err := m.EnsureRunning(ctx); err != nil {
		return err
	}

	if spec.IntraOpThreads == 0 && m.cfg.IntraOpThreads > 0 {
		spec.IntraOpThreads = int32(m.cfg.IntraOpThreads)
	}
	if len(spec.Providers) == 0 && len(m.cfg.Providers) > 0 {
		spec.Providers = append([]string{}, m.cfg.Providers...)
	}

	if loaded, err := m.stub.ListLoadedModels(ctx, &pb.ListLoadedModelsRequest{}); err == nil {
		for _, model := range loaded.Models {
			if model.ModelId == spec.ModelId && model.ModelName == spec.ModelName {
				return nil
			}
		}
	}

	_, err := m.stub.LoadModel(ctx, &pb.LoadModelRequest{Spec: &spec})
	return err
}

func (m *Manager) RunJob(ctx context.Context, jobID, inputPath, outputDir string, params map[string]string) (*pb.JobStatus, error) {
	m.jobMu.Lock()
	defer m.jobMu.Unlock()

	_, err := m.stub.StartJob(ctx, &pb.StartJobRequest{
		JobId:     jobID,
		InputPath: inputPath,
		OutputDir: outputDir,
		Params:    params,
	})
	if err != nil {
		return nil, err
	}

	stream, err := m.stub.StreamJobStatus(ctx, &pb.StreamJobStatusRequest{JobId: jobID})
	if err != nil {
		return nil, err
	}

	var final *pb.JobStatus
	for {
		status, recvErr := stream.Recv()
		if recvErr != nil {
			if final != nil {
				return final, nil
			}
			return nil, recvErr
		}
		final = status
		if status.State == pb.JobState_JOB_STATE_COMPLETED ||
			status.State == pb.JobState_JOB_STATE_FAILED ||
			status.State == pb.JobState_JOB_STATE_CANCELLED {
			return status, nil
		}
	}
}

func (m *Manager) StopJob(ctx context.Context, jobID string) {
	if m.stub == nil {
		return
	}
	_, _ = m.stub.StopJob(ctx, &pb.StopJobRequest{JobId: jobID})
}

func (m *Manager) startProcess() error {
	if len(m.cfg.Command) == 0 {
		return fmt.Errorf("diarization engine command is not configured")
	}
	if m.cmd != nil && m.cmd.Process != nil {
		return nil
	}

	cmd := exec.Command(m.cfg.Command[0], m.cfg.Command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "DIAR_ENGINE_SOCKET="+m.cfg.SocketPath)

	logger.Info("Starting diarization engine daemon", "command", strings.Join(m.cfg.Command, " "))
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start diarization engine: %w", err)
	}
	m.cmd = cmd

	go func() {
		err := cmd.Wait()
		if err != nil {
			logger.Warn("Diarization engine process exited", "error", err)
		}
	}()

	return nil
}

func (m *Manager) dial(ctx context.Context) error {
	if m.conn != nil {
		return nil
	}

	dialer := func(ctx context.Context, addr string) (net.Conn, error) {
		if strings.HasPrefix(addr, "unix:") {
			return net.Dial("unix", strings.TrimPrefix(addr, "unix:"))
		}
		return net.Dial("tcp", addr)
	}

	target := m.cfg.SocketPath
	if !strings.HasPrefix(target, "unix:") {
		target = "unix:" + target
	}

	conn, err := grpc.DialContext(
		ctx,
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
	)
	if err != nil {
		return err
	}
	m.conn = conn
	m.stub = pb.NewAsrEngineClient(conn)
	return nil
}

func (m *Manager) ping(ctx context.Context) error {
	if m.stub == nil {
		return fmt.Errorf("diarization engine not connected")
	}
	_, err := m.stub.GetEngineInfo(ctx, &pb.GetEngineInfoRequest{})
	return err
}

func (m *Manager) closeConn() {
	if m.conn != nil {
		_ = m.conn.Close()
	}
	m.conn = nil
	m.stub = nil
}

func getenv(key, fallback string) string {
	if val := strings.TrimSpace(os.Getenv(key)); val != "" {
		return val
	}
	return fallback
}
