# Automatic Audio Title Generation Sprint

## Goal

After automatic summarization completes, generate a concise recording title with the configured small LLM model and persist it to the audio file row. The Home page must update reactively through SSE as soon as the title is saved.

## Product Rules

- Run only after an automatic summary reaches `completed`.
- Use the active LLM provider's configured small model.
- If no LLM provider or small model is configured, do nothing and do not fail transcription or summarization.
- The generated title must describe the audio accurately in no more than 7 words.
- Enforce the 7-word limit in both the prompt and backend validation/post-processing.
- Do not rename when the provider returns an empty, generic, or invalid title.
- Persist the title on the existing recording/file record.
- Track title-generation metadata on the recording so this runs at most once per audio file.
- Publish a file SSE event after persistence so the Home page refreshes without a manual reload.

## Data Model

Add explicit metadata columns to `transcriptions`:

- `llm_title_generated` boolean, default `false`
- `llm_title_generated_at` nullable timestamp

Rules:
- If `llm_title_generated = true`, skip automatic title generation for that recording.
- Set `llm_title_generated = true` after a successful LLM-generated rename.
- Leave `llm_title_generated = false` when the provider is not configured, provider output is invalid, or generation fails. This allows a future completed summary workflow to try again only if the feature is intentionally re-triggered.
- Manual user title edits should not reset this metadata unless a later product decision explicitly asks for re-generation.

## Sprint Run 1: Backend Title Generator

Scope:
- Add the DB migration/model fields for title-generation metadata.
- Add a small title-generation step after `summary.completed` in the summarization workflow.
- Skip immediately when `llm_title_generated` is already true for the transcription job.
- Reuse the existing LLM config lookup, small-model client construction, and summary context.
- Prompt for one title only, no quotes, no markdown, no punctuation-heavy output, maximum 7 words.
- Add a backend sanitizer that trims quotes, collapses whitespace, removes trailing punctuation, and rejects or clips overlong outputs according to the 7-word rule.
- Update the `transcriptions.title`, `llm_title_generated`, and `llm_title_generated_at` fields through the job repository or a focused file-title helper in one DB update.
- Publish `file.updated` with `id`, `title`, and current status after the DB update.

Tests:
- Missing LLM provider/small model is a no-op.
- Already-renamed recordings are skipped without calling the provider.
- Completed summary triggers a title update when the provider returns a valid title.
- Provider output over 7 words is enforced before persistence.
- Empty/generic provider output leaves the existing title unchanged.
- Successful rename marks the metadata fields.
- Title update publishes a file event.

## Sprint Run 2: Frontend Reactivity Check

Scope:
- Keep the Home page driven by `useFiles` and existing `/api/v1/events` SSE invalidation.
- Ensure `file.updated` is treated as a file event by `useFileEvents`.
- Verify the audio detail title also updates via existing file query invalidation when relevant.
- No new UI controls are needed for this sprint.

Tests:
- Browser check: after summary completion and automatic rename, the Home page list shows the new title without refresh.
- Existing manual title editing still works.
- Existing SSE reconnection behavior remains unchanged.

## Sprint Run 3: Verification and Commit

Scope:
- Run focused backend tests for summarization/title generation and event publishing.
- Run frontend type-check/build if frontend event code changes.
- Run `git diff --check`.
- Commit backend generator and frontend/event polish atomically.

Risks:
- Provider title quality may vary; backend validation should prefer leaving the title unchanged over saving a bad title.
- This should not create a second durable queue. Keep it as a lightweight post-summary action unless title generation needs retries later.
