# Backend Architecture Review Remediation Sprint 04 LLM Secrets

Date: 2026-05-03

## Scope

Sprint 4 addresses Finding 2: saved LLM provider API keys were stored raw in `llm_profiles.config_json`.

## Changes

- Added `llmprovider.ProtectedRepository`, a repository wrapper that encrypts `LLMConfig.APIKey` before persistence and decrypts it on reads.
- Kept the existing schema and `config_json.api_key` location to avoid a migration in this remediation sprint.
- Added `LLM_CREDENTIAL_SECRET` config support, defaulting to the JWT secret when unset.
- Wired the protected repository through app and API test composition so chat, summarization, automation, account, and LLM provider services receive decrypted runtime configs.
- Preserved backward compatibility for existing plaintext `config_json.api_key` rows.
- Updated API regression coverage to assert the saved row no longer contains the raw key and the response still redacts the key.

## TDD Evidence

The new tests failed before implementation:

```txt
undefined: NewProtectedRepository
```

After implementation, the storage/API assertions passed and persisted keys use the `enc:v1:` marker instead of raw plaintext.

## Verification

Passed:

```sh
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/llmprovider -run 'TestProtectedRepository'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestLLMProviderSettingsSaveTestsConnectionAndMasksKey'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/config ./internal/database ./internal/models ./internal/repository ./internal/llmprovider
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/api -run 'TestLLMProvider|TestSecurity'
GOCACHE=/private/tmp/scriberr-go-cache go test ./internal/app ./cmd/server
git diff --check
```

## Notes

The LLM provider/API commands that use `httptest.NewServer` required loopback permission because sandboxed binds are denied in this environment.

Existing plaintext keys remain readable so installs can continue operating. A future migration can rotate those rows by saving the config through the protected repository.
