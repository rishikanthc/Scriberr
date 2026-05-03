package automation

import (
	"context"
	"testing"

	filesdomain "scriberr/internal/files"
	"scriberr/internal/models"
	transcriptiondomain "scriberr/internal/transcription"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestFileReadyAutoTranscribesWhenEnabled(t *testing.T) {
	file := &models.TranscriptionJob{ID: "file-1", UserID: 7, Status: models.StatusUploaded}
	user := &models.User{ID: 7, AutoTranscriptionEnabled: true}
	profile := &models.TranscriptionProfile{ID: "profile-1", UserID: 7}
	transcriptions := &fakeTranscriptionCreator{job: &models.TranscriptionJob{ID: "tr-1", UserID: 7, Status: models.StatusPending, SourceFileHash: &file.ID}}
	events := &fakeAutomationEvents{}

	service := NewService(
		&fakeAutomationFiles{file: file},
		&fakeAutomationUsers{user: user},
		&fakeAutomationProfiles{profile: profile},
		nil,
		transcriptions,
	)
	service.SetEventPublisher(events)

	require.NoError(t, service.FileReady(context.Background(), filesdomain.ReadyEvent{FileID: file.ID, Kind: "audio", Status: "ready"}))

	require.Len(t, transcriptions.commands, 1)
	require.Equal(t, uint(7), transcriptions.commands[0].UserID)
	require.Equal(t, file.ID, transcriptions.commands[0].FileID)
	require.Len(t, events.transcriptionEvents, 1)
	require.Equal(t, "transcription.created", events.transcriptionEvents[0].name)
}

func TestFileReadyNoOpsWhenRuntimeConfigurationIsMissing(t *testing.T) {
	file := &models.TranscriptionJob{ID: "file-1", UserID: 7, Status: models.StatusUploaded}
	user := &models.User{ID: 7, AutoTranscriptionEnabled: true, AutoRenameEnabled: true}
	transcriptions := &fakeTranscriptionCreator{job: &models.TranscriptionJob{ID: "tr-1"}}

	service := NewService(
		&fakeAutomationFiles{file: file},
		&fakeAutomationUsers{user: user},
		&fakeAutomationProfiles{err: gorm.ErrRecordNotFound},
		&fakeAutomationLLM{err: gorm.ErrRecordNotFound},
		transcriptions,
	)

	require.NoError(t, service.FileReady(context.Background(), filesdomain.ReadyEvent{FileID: file.ID, Kind: "audio", Status: "ready"}))
	require.Empty(t, transcriptions.commands)
}

func TestFileReadySkipsWhenTranscriptionAlreadyExists(t *testing.T) {
	file := &models.TranscriptionJob{ID: "file-1", UserID: 7, Status: models.StatusUploaded}
	user := &models.User{ID: 7, AutoTranscriptionEnabled: true}
	transcriptions := &fakeTranscriptionCreator{job: &models.TranscriptionJob{ID: "tr-1"}}

	service := NewService(
		&fakeAutomationFiles{file: file, transcriptionCount: 1},
		&fakeAutomationUsers{user: user},
		&fakeAutomationProfiles{profile: &models.TranscriptionProfile{ID: "profile-1"}},
		nil,
		transcriptions,
	)

	require.NoError(t, service.FileReady(context.Background(), filesdomain.ReadyEvent{FileID: file.ID, Kind: "audio", Status: "ready"}))
	require.Empty(t, transcriptions.commands)
}

func TestFileReadyNoOpsWhenEventDoesNotReferenceReadyFile(t *testing.T) {
	transcriptions := &fakeTranscriptionCreator{job: &models.TranscriptionJob{ID: "tr-1"}}
	service := NewService(
		&fakeAutomationFiles{err: gorm.ErrRecordNotFound},
		&fakeAutomationUsers{user: &models.User{ID: 7, AutoTranscriptionEnabled: true}},
		&fakeAutomationProfiles{profile: &models.TranscriptionProfile{ID: "profile-1"}},
		nil,
		transcriptions,
	)

	require.NoError(t, service.FileReady(context.Background(), filesdomain.ReadyEvent{FileID: "transcription-1", Kind: "audio", Status: "ready"}))
	require.Empty(t, transcriptions.commands)
}

type fakeAutomationFiles struct {
	file               *models.TranscriptionJob
	transcriptionCount int64
	err                error
}

func (f *fakeAutomationFiles) FindReadyFileByID(context.Context, string) (*models.TranscriptionJob, error) {
	return f.file, f.err
}

func (f *fakeAutomationFiles) CountTranscriptionsBySourceFile(context.Context, uint, string) (int64, error) {
	return f.transcriptionCount, f.err
}

type fakeAutomationUsers struct {
	user *models.User
	err  error
}

func (f *fakeAutomationUsers) FindAutomationUserByID(context.Context, uint) (*models.User, error) {
	return f.user, f.err
}

type fakeAutomationProfiles struct {
	profile *models.TranscriptionProfile
	err     error
}

func (f *fakeAutomationProfiles) FindDefaultByUser(context.Context, uint) (*models.TranscriptionProfile, error) {
	return f.profile, f.err
}

type fakeAutomationLLM struct {
	config *models.LLMConfig
	err    error
}

func (f *fakeAutomationLLM) GetActiveByUser(context.Context, uint) (*models.LLMConfig, error) {
	return f.config, f.err
}

type fakeTranscriptionCreator struct {
	job      *models.TranscriptionJob
	err      error
	commands []transcriptiondomain.CreateCommand
}

func (f *fakeTranscriptionCreator) Create(_ context.Context, cmd transcriptiondomain.CreateCommand) (*models.TranscriptionJob, error) {
	f.commands = append(f.commands, cmd)
	return f.job, f.err
}

type fakeAutomationEvents struct {
	transcriptionEvents []fakeTranscriptionEvent
}

type fakeTranscriptionEvent struct {
	name            string
	transcriptionID string
	payload         map[string]any
}

func (f *fakeAutomationEvents) PublishTranscriptionEvent(_ context.Context, name string, transcriptionID string, payload map[string]any) {
	f.transcriptionEvents = append(f.transcriptionEvents, fakeTranscriptionEvent{name: name, transcriptionID: transcriptionID, payload: payload})
}
