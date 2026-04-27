# Scriberr Frontend Architecture Rules

These are the React rules for the revamped Scriberr app in `web/frontend`. They describe the target architecture, not every legacy file currently present. The product is a logged-in transcription workspace with uploads, durable progress, transcript playback, notes, chat, settings, and model/profile configuration. Optimize for code that is easy to find, easy to read, and hard to misuse.

## 1. Organize by Product Feature

Scriberr frontend work should live by feature:

```txt
features/auth
features/files
features/home
features/settings
features/transcription
```

Use this placement rule:

```txt
feature-specific and domain-aware -> features/<feature>
visual primitive with no domain knowledge -> components/ui or shared/ui
cross-route app shell -> components/layout or contexts
generic browser/client utility -> lib or utils
```

Avoid adding new root-level domain components for transcription, files, or settings. Existing files can be moved as part of revamp work. The desired end state is that a developer can open a feature folder and find its API calls, hooks, components, and local types.

## 2. Pages Orchestrate, Children Render

Route/page components should coordinate route params, navigation, queries, and high-level workflow state. Child components should render focused parts of the workflow.

Preferred split:

```txt
page/view component       -> route params, navigation, high-level hooks
feature hook              -> server data, mutations, event subscriptions
panel/section component   -> layout and user workflow
leaf component            -> display and local interaction
ui primitive              -> styling and accessibility behavior
```

Avoid components that fetch data, mutate global state, manage resize/drag/drop, own many dialogs, and render dense UI all at once. When a screen grows, extract by user workflow: upload progress, transcript section, notes sidebar, summary dialog, execution dialog, table actions, chat side panel.

## 3. Server State Lives in TanStack Query

Backend data is not local React state. Files, transcriptions, profiles, transcripts, logs, executions, notes, summaries, API keys, settings, and model capabilities should be fetched through typed API functions and exposed through feature hooks backed by TanStack Query.

Rules:

- API modules contain endpoint URLs, request payloads, response types, and response normalization.
- Hooks define query keys, enabled conditions, polling, invalidation, and mutations.
- Components consume hooks instead of assembling endpoint URLs inline.
- Mutations invalidate the smallest useful query keys.
- Query keys are exported constants when reused by events, mutations, or sibling hooks.

Continue consolidating older direct fetches into `features/*/api` plus `features/*/hooks`.

## 4. Global State Is Rare

Use the lowest-power state owner:

```txt
local state        -> dialogs, edit mode, player time, hover, selected tab
URL state          -> selected resource, filters, search, pagination, shareable view state
TanStack Query     -> backend resources and async mutations
Zustand/context    -> auth session, theme, app-wide upload workflow, cross-route event channels
refs               -> imperative media players, drag counters, DOM measurement
```

Do not promote state because prop passing feels mildly inconvenient. Promote it only when distant parts of the app coordinate the same client-only workflow.

## 5. Effects Synchronize With External Systems

Effects are appropriate for SSE connections, window event listeners, media APIs, drag/drop listeners, timers, and imperative DOM integrations. They are not for derived values or routine data fetching.

Good Scriberr effects:

- Connect to `/api/v1/events` and invalidate query caches.
- Add global drag/drop listeners for upload surfaces.
- Track resize gestures for a transcript/chat split view.
- Synchronize media player callbacks with current playback time.

Avoid effects for:

- Computing display labels.
- Copying query data into local state without a user edit reason.
- Fetching directly instead of using query hooks.
- Resetting state that can be derived from route params or query keys.

Every subscription effect must have complete cleanup.

## 6. Long-Running Work Is a First-Class UI Model

Scriberr actions often start backend workflows that continue after the request returns. The UI should model that honestly.

Rules:

- Uploads need pending, success, error, retry, and dismiss behavior.
- Transcriptions need queued, processing, completed, failed, and canceled states.
- Events should make the UI feel live; queries remain the source of truth.
- Poll only active queued/processing resources and stop polling terminal states.
- A page must be correct after refresh even if no SSE event was received.

Use events for fast feedback and queries for truth.

## 7. Protect Transcript and Audio Hot Paths

The audio detail screen can become expensive: media playback, current-time updates, transcript highlighting, words/segments, notes, chat, menus, downloads, logs, summaries, and speaker editing can all trigger renders.

Performance rules:

- Do not re-render the whole detail page on every audio time tick if only highlighting needs it.
- Keep current-time subscriptions close to transcript/player components.
- Memoize expensive transcript normalization, speaker mapping, and search/highlight calculations when inputs are stable.
- Virtualize or window transcript rendering if segment or word counts become large.
- Keep heavy dialog content conditionally mounted.
- Prefer primitive props and stable callbacks for dense transcript rows.
- Avoid parsing large transcript JSON repeatedly during render; normalize in the API/hook layer or a memoized selector.

Measure before broad memoization, but keep ownership boundaries narrow enough that optimization is possible without rewriting the screen.

## 8. Type API Contracts at the Boundary

The backend uses snake_case fields, public IDs such as `file_...` and `tr_...`, structured errors, cursor pagination, and durable status fields. The frontend should make those contracts explicit.

Rules:

- Define response and payload types beside the API function that uses them.
- Keep backend field names in API types unless a feature intentionally normalizes them.
- Normalize API quirks in API functions or hooks, not leaf UI components.
- Parse structured error responses into useful messages.
- Avoid `any`; if a response is unknown, type it as `unknown` and narrow it.
- Keep statuses as string unions, not loose strings.

Compile-time friction at the API boundary is good developer UX.

## 9. Use the Design System Vocabulary

Scriberr has shadcn-style primitives, local shared UI, Lucide icons, Manrope, and design tokens. Use those first.

Rules:

- Use `components/ui` primitives for buttons, dialogs, dropdowns, inputs, tabs, tooltips, tables, progress, sliders, switches, and popovers.
- Use Lucide icons for recognizable actions.
- Use design tokens for surfaces, text, borders, radius, shadows, status colors, and brand colors.
- Do not create feature-specific button/modal/input styling unless primitives cannot express the workflow.
- Keep operational screens quiet, dense, scannable, and predictable.

Cards are for repeated items, dialogs, and contained tools. Avoid nesting cards inside cards or wrapping every page section in decorative panels.

## 10. Accessibility Is Part of the Component API

Scriberr includes audio controls, tables, dropdowns, dialogs, drag/drop, transcript selection, and resizable panels. These workflows must work without relying only on pointer interaction or color.

Rules:

- Buttons are real buttons.
- Inputs have labels or accessible names.
- Icon-only controls have tooltips or screen-reader labels.
- Dialogs trap focus and return focus.
- Dropdown actions are keyboard reachable.
- Color is not the only status signal.
- Loading, empty, and error states are explicit.
- Audio and transcript actions remain usable on mobile.

If a component cannot be used from the keyboard, its API is incomplete.

## 11. Mobile and Desktop Are Both Primary

Build responsive layouts deliberately.

Rules:

- Use mobile-aware hooks only when CSS is not enough.
- Keep transcript, player, side panels, and upload actions reachable on small screens.
- Avoid fixed widths without min/max constraints.
- Ensure long titles, filenames, model names, and error messages truncate or wrap intentionally.
- Do not let sticky headers, players, or floating menus cover transcript content.

Desktop may show split panes; mobile should favor one focused workflow at a time.

## 12. Prefer Explicit Developer UX

Human-readable frontend code in this repo has:

```txt
typed props
named event handlers
feature-level hooks
small API modules
clear query keys
explicit loading/error/empty states
predictable file placement
minimal global state
few effects
stable visual primitives
```

Prefer obvious names over clever abstractions. A future developer should understand a workflow by reading the feature folder, not by tracing global utilities.

## 13. Tests Protect User Workflows

Prioritize tests around user-visible risks:

- Auth/session initialization and protected routes.
- Upload and import flows.
- Transcription create/cancel/progress states.
- Transcript rendering and player seeking.
- Notes, summaries, downloads, speaker rename, and chat entry points.
- Settings/profile forms and validation errors.
- Empty, loading, failed, and unauthorized states.

Use mocked API boundaries for component and hook tests. Use browser-level tests for core flows once stable fixtures exist.

## 14. Review Checklist

Use this checklist for frontend changes:

```txt
[ ] Does the code live in the right feature/shared boundary?
[ ] Is server state handled through API functions and query hooks?
[ ] Are query keys and invalidation scoped correctly?
[ ] Is client state kept as local as possible?
[ ] Are effects only used for external synchronization?
[ ] Are loading, empty, error, and terminal states handled?
[ ] Does the UI work on mobile and desktop?
[ ] Are controls accessible and keyboard reachable?
[ ] Are hot transcript/audio paths protected from avoidable re-renders?
[ ] Are API responses typed and errors surfaced clearly?
[ ] Does the component read like a workflow a user actually performs?
```

## Strong Default Philosophy

Build Scriberr's frontend like this:

```txt
Feature-owned workflows.
Typed API boundaries.
Server state in queries.
Events for freshness, queries for truth.
Local state by default.
Small render components.
Responsive operational UI.
Accessible controls.
Measured transcript/audio performance.
Design primitives before custom styling.
```

The best frontend code in this repo should feel direct, predictable, and hard to misuse.
