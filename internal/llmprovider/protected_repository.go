package llmprovider

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"

	"scriberr/internal/models"
	"scriberr/internal/repository"
)

const encryptedAPIKeyPrefix = "enc:v1:"

type ProtectedRepository struct {
	inner repository.LLMConfigRepository
	aead  cipher.AEAD
}

func NewProtectedRepository(inner repository.LLMConfigRepository, secret string) (*ProtectedRepository, error) {
	if inner == nil {
		return nil, fmt.Errorf("llm config repository is required")
	}
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("llm credential secret is required")
	}
	key := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &ProtectedRepository{inner: inner, aead: aead}, nil
}

func MustProtectedRepository(inner repository.LLMConfigRepository, secret string) repository.LLMConfigRepository {
	protected, err := NewProtectedRepository(inner, secret)
	if err != nil {
		panic(err)
	}
	return protected
}

func (r *ProtectedRepository) Create(ctx context.Context, entity *models.LLMConfig) error {
	config, err := r.encryptConfig(entity)
	if err != nil {
		return err
	}
	return r.inner.Create(ctx, config)
}

func (r *ProtectedRepository) FindByID(ctx context.Context, id interface{}) (*models.LLMConfig, error) {
	config, err := r.inner.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return r.decryptConfig(config), nil
}

func (r *ProtectedRepository) Update(ctx context.Context, entity *models.LLMConfig) error {
	config, err := r.encryptConfig(entity)
	if err != nil {
		return err
	}
	return r.inner.Update(ctx, config)
}

func (r *ProtectedRepository) Delete(ctx context.Context, id interface{}) error {
	return r.inner.Delete(ctx, id)
}

func (r *ProtectedRepository) List(ctx context.Context, offset, limit int) ([]models.LLMConfig, int64, error) {
	configs, count, err := r.inner.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	for i := range configs {
		configs[i] = *r.decryptConfig(&configs[i])
	}
	return configs, count, nil
}

func (r *ProtectedRepository) GetActive(ctx context.Context) (*models.LLMConfig, error) {
	config, err := r.inner.GetActive(ctx)
	if err != nil {
		return nil, err
	}
	return r.decryptConfig(config), nil
}

func (r *ProtectedRepository) GetActiveByUser(ctx context.Context, userID uint) (*models.LLMConfig, error) {
	config, err := r.inner.GetActiveByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return r.decryptConfig(config), nil
}

func (r *ProtectedRepository) ReplaceActiveByUser(ctx context.Context, userID uint, config *models.LLMConfig) error {
	protected, err := r.encryptConfig(config)
	if err != nil {
		return err
	}
	return r.inner.ReplaceActiveByUser(ctx, userID, protected)
}

func (r *ProtectedRepository) encryptConfig(config *models.LLMConfig) (*models.LLMConfig, error) {
	if config == nil || config.APIKey == nil {
		return config, nil
	}
	value := strings.TrimSpace(*config.APIKey)
	if value == "" || strings.HasPrefix(value, encryptedAPIKeyPrefix) {
		return config, nil
	}
	next := *config
	encrypted, err := r.encrypt(value)
	if err != nil {
		return nil, err
	}
	next.APIKey = &encrypted
	return &next, nil
}

func (r *ProtectedRepository) decryptConfig(config *models.LLMConfig) *models.LLMConfig {
	if config == nil || config.APIKey == nil {
		return config
	}
	value := strings.TrimSpace(*config.APIKey)
	if value == "" || !strings.HasPrefix(value, encryptedAPIKeyPrefix) {
		return config
	}
	next := *config
	decrypted, err := r.decrypt(value)
	if err != nil {
		return config
	}
	next.APIKey = &decrypted
	return &next
}

func (r *ProtectedRepository) encrypt(plaintext string) (string, error) {
	nonce := make([]byte, r.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := r.aead.Seal(nil, nonce, []byte(plaintext), nil)
	return encryptedAPIKeyPrefix +
		base64.RawURLEncoding.EncodeToString(nonce) + ":" +
		base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (r *ProtectedRepository) decrypt(value string) (string, error) {
	encoded := strings.TrimPrefix(value, encryptedAPIKeyPrefix)
	nonceText, cipherText, ok := strings.Cut(encoded, ":")
	if !ok {
		return "", fmt.Errorf("invalid encrypted llm credential")
	}
	nonce, err := base64.RawURLEncoding.DecodeString(nonceText)
	if err != nil {
		return "", err
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	plaintext, err := r.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
