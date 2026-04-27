# LLM Provider Settings Sprint

Date: 2026-04-27

## Goal

Add a settings tab where an authenticated user can configure one active OpenAI-compatible LLM provider endpoint with an optional API key. Saving must verify the provider is reachable and can list models before persisting the configuration.

## Scope

- Backend API for reading and saving the active LLM provider.
- Connection testing for OpenAI-compatible `/models` endpoints, including OpenAI, llama.cpp, and Ollama's OpenAI-compatible API.
- Frontend settings tab using feature-local API functions and TanStack Query hooks.
- Compact settings UI aligned with the current Scriberr design system.

## API Plan

- `GET /api/v1/settings/llm-provider`
  - Returns configured base URL, provider label, key presence, model count, and last tested time.
  - Does not return the raw API key.
- `PUT /api/v1/settings/llm-provider`
  - Accepts `base_url` and optional `api_key`.
  - Normalizes and validates the base URL.
  - Calls the provider's model-list endpoint before saving.
  - Stores only after a successful connection test.

## Frontend Plan

- Add `LLM Providers` as a settings tab.
- Add two fields:
  - Base endpoint URL.
  - Optional API key.
- Show the saved key as present without exposing it.
- Use a single Save action that tests the connection through the backend before persisting.
- Surface loading, error, empty, and success states without introducing page-level decoration.

## Verification

- Backend tests for auth, validation, connection failure, successful save, API-key masking, and route contract coverage.
- Frontend production build.
- `git diff --check` before each commit.

## Commit Plan

1. Document sprint plan.
2. Add backend LLM provider settings API and tests.
3. Add frontend LLM Providers settings tab.
