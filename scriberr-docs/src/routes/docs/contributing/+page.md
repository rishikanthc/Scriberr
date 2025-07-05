# Contributing to Scriberr

Thank you for your interest in contributing to Scriberr! This guide will help you get started with contributing to the project.

## Getting Started

### Prerequisites

Before contributing, make sure you have:

- **Go**: Version 1.21 or later
- **Node.js**: Version 18 or later
- **Git**: Latest version
- **Docker**: For testing (optional)

### Development Setup

1. **Fork the Repository**:
   ```bash
   # Fork on GitHub first, then clone your fork
   git clone https://github.com/your-username/scriberr.git
   cd scriberr
   
   # Add upstream remote
   git remote add upstream https://github.com/noeticgeek/scriberr.git
   ```

2. **Set Up Development Environment**:
   ```bash
   # Install Go dependencies
   go mod download
   
   # Set up frontend
   cd scriberr-frontend
   npm install
   cd ..
   ```

3. **Create a Feature Branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Contribution Areas

### Backend Development (Go)

The backend is written in Go and handles:

- **API Endpoints**: REST API for transcription and management
- **Audio Processing**: Integration with WhisperX
- **Database Operations**: SQLite database management
- **Background Tasks**: Job queue and processing

#### Key Directories

- `cmd/`: Application entry points
- `internal/`: Internal packages
  - `handlers/`: HTTP request handlers
  - `database/`: Database operations
  - `models/`: Data structures
  - `middleware/`: HTTP middleware
  - `tasks/`: Background task processing

#### Development Guidelines

```go
// Example handler structure
package handlers

import (
    "net/http"
    "encoding/json"
)

type TranscriptionRequest struct {
    AudioFile string `json:"audio_file"`
    ModelSize string `json:"model_size"`
}

func HandleTranscription(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### Frontend Development (SvelteKit)

The frontend is built with SvelteKit and provides:

- **User Interface**: Modern, responsive web interface
- **Audio Management**: File upload and recording
- **Real-time Updates**: WebSocket integration
- **Export Features**: Multiple format support

#### Key Directories

- `src/routes/`: Page components
- `src/lib/components/`: Reusable UI components
- `src/lib/stores/`: State management
- `src/lib/utils/`: Utility functions

#### Development Guidelines

```svelte
<!-- Example component structure -->
<script lang="ts">
  import { Button } from '$lib/components/ui/button';
  
  interface Props {
    title: string;
    onAction: () => void;
  }
  
  let { title, onAction } = $props<Props>();
</script>

<div class="component">
  <h2>{title}</h2>
  <Button on:click={onAction}>Action</Button>
</div>
```

### Documentation

Help improve documentation:

- **User Guides**: Usage instructions and tutorials
- **API Documentation**: Backend API reference
- **Code Comments**: Inline code documentation
- **README Updates**: Project overview and setup

## Development Workflow

### 1. Issue Discussion

Before starting work:

1. **Check Existing Issues**: Search for similar issues
2. **Create New Issue**: Describe the problem or feature
3. **Discuss Approach**: Get feedback on your solution
4. **Get Assigned**: Wait for issue assignment

### 2. Development

```bash
# Keep your fork updated
git fetch upstream
git checkout main
git merge upstream/main

# Create feature branch
git checkout -b feature/your-feature-name

# Make your changes
# ... edit files ...

# Test your changes
go test ./...
cd scriberr-frontend && npm test && cd ..
```

### 3. Testing

#### Backend Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/handlers -v

# Run integration tests
go test -tags=integration ./...
```

#### Frontend Testing

```bash
cd scriberr-frontend

# Run unit tests
npm test

# Run tests with coverage
npm run test:coverage

# Run end-to-end tests
npm run test:e2e
```

### 4. Code Quality

#### Go Code Style

```bash
# Format code
gofmt -s -w .

# Run linter
golangci-lint run

# Check for security issues
gosec ./...

# Run static analysis
staticcheck ./...
```

#### Frontend Code Style

```bash
cd scriberr-frontend

# Format code
npm run format

# Lint code
npm run lint

# Type checking
npm run check
```

### 5. Commit and Push

```bash
# Stage changes
git add .

# Commit with descriptive message
git commit -m "feat: add new transcription feature

- Add support for batch processing
- Implement progress tracking
- Update API documentation

Closes #123"

# Push to your fork
git push origin feature/your-feature-name
```

### 6. Pull Request

1. **Create Pull Request**: Use GitHub's PR interface
2. **Fill Template**: Complete the PR template
3. **Link Issues**: Reference related issues
4. **Request Review**: Ask for code review
5. **Address Feedback**: Respond to review comments

## Code Standards

### Go Standards

- **Formatting**: Use `gofmt` or `goimports`
- **Naming**: Follow Go naming conventions
- **Error Handling**: Always check and handle errors
- **Documentation**: Add comments for exported functions
- **Testing**: Write tests for new functionality

```go
// Example of good Go code
package handlers

import (
    "context"
    "encoding/json"
    "net/http"
)

// TranscriptionHandler handles audio transcription requests
type TranscriptionHandler struct {
    service TranscriptionService
}

// HandleTranscription processes audio transcription requests
func (h *TranscriptionHandler) HandleTranscription(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()
    
    var req TranscriptionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid request", http.StatusBadRequest)
        return
    }
    
    result, err := h.service.Transcribe(ctx, req)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    json.NewEncoder(w).Encode(result)
}
```

### Frontend Standards

- **TypeScript**: Use TypeScript for type safety
- **Components**: Create reusable, composable components
- **State Management**: Use Svelte stores appropriately
- **Styling**: Use Tailwind CSS classes
- **Accessibility**: Follow WCAG guidelines

```svelte
<!-- Example of good Svelte component -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { Button } from '$lib/components/ui/button';
  
  interface Props {
    title: string;
    disabled?: boolean;
  }
  
  let { title, disabled = false } = $props<Props>();
  const dispatch = createEventDispatcher<{
    action: { value: string };
  }>();
  
  function handleClick() {
    dispatch('action', { value: 'clicked' });
  }
</script>

<div class="flex flex-col gap-4 p-4">
  <h2 class="text-xl font-semibold">{title}</h2>
  <Button 
    disabled={disabled}
    on:click={handleClick}
  >
    Action
  </Button>
</div>
```

## Testing Guidelines

### Backend Testing

- **Unit Tests**: Test individual functions and methods
- **Integration Tests**: Test API endpoints and database operations
- **Mocking**: Use mocks for external dependencies
- **Test Coverage**: Aim for 80%+ coverage

```go
// Example test
func TestTranscriptionHandler_HandleTranscription(t *testing.T) {
    // Setup
    service := &MockTranscriptionService{}
    handler := &TranscriptionHandler{service: service}
    
    // Test cases
    tests := []struct {
        name     string
        request  TranscriptionRequest
        expected int
    }{
        {"valid request", TranscriptionRequest{AudioFile: "test.wav"}, http.StatusOK},
        {"invalid request", TranscriptionRequest{}, http.StatusBadRequest},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Frontend Testing

- **Component Tests**: Test component behavior and rendering
- **Integration Tests**: Test user workflows
- **Accessibility Tests**: Ensure accessibility compliance
- **Visual Regression**: Test UI consistency

```typescript
// Example test
import { render, screen } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import TranscriptionForm from './TranscriptionForm.svelte';

describe('TranscriptionForm', () => {
  it('should render upload button', () => {
    render(TranscriptionForm);
    expect(screen.getByRole('button', { name: /upload/i })).toBeInTheDocument();
  });
  
  it('should handle file upload', async () => {
    // Test implementation
  });
});
```

## Documentation Standards

### Code Documentation

- **Package Comments**: Document package purpose
- **Function Comments**: Explain complex functions
- **Type Comments**: Document custom types
- **Example Usage**: Provide usage examples

```go
// Package handlers provides HTTP request handlers for the Scriberr API.
package handlers

// TranscriptionRequest represents a request to transcribe audio.
type TranscriptionRequest struct {
    AudioFile string `json:"audio_file" validate:"required"`
    ModelSize string `json:"model_size" validate:"oneof=tiny base small medium large"`
}

// HandleTranscription processes audio transcription requests.
// It validates the request, processes the audio file, and returns
// the transcription result.
func HandleTranscription(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

### User Documentation

- **Clear Structure**: Use consistent headings and organization
- **Code Examples**: Provide working code examples
- **Screenshots**: Include relevant screenshots
- **Step-by-step**: Break complex processes into steps

## Review Process

### Pull Request Review

1. **Automated Checks**: CI/CD pipeline runs tests
2. **Code Review**: Maintainers review the code
3. **Testing**: Verify functionality works as expected
4. **Documentation**: Ensure documentation is updated
5. **Approval**: Get approval from maintainers

### Review Checklist

- [ ] Code follows project standards
- [ ] Tests are included and passing
- [ ] Documentation is updated
- [ ] No breaking changes (or properly documented)
- [ ] Performance impact is considered
- [ ] Security implications are addressed

## Release Process

### Version Management

- **Semantic Versioning**: Follow semver.org guidelines
- **Changelog**: Update CHANGELOG.md with changes
- **Release Notes**: Create detailed release notes
- **Tagging**: Tag releases in Git

### Release Checklist

- [ ] All tests passing
- [ ] Documentation updated
- [ ] Changelog updated
- [ ] Version bumped
- [ ] Release notes prepared
- [ ] Assets built and uploaded

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and discussions
- **Pull Requests**: Code review and feedback
- **Email**: For sensitive or private matters

### Resources

- **Go Documentation**: https://golang.org/doc/
- **SvelteKit Documentation**: https://kit.svelte.dev/docs
- **Tailwind CSS**: https://tailwindcss.com/docs
- **shadcn-svelte**: https://www.shadcn-svelte.com/

## Recognition

### Contributors

All contributors are recognized in:

- **README.md**: List of contributors
- **GitHub Contributors**: Automatic contributor graph
- **Release Notes**: Credit for significant contributions
- **Documentation**: Attribution in documentation

### Types of Contributions

- **Code**: Bug fixes, features, improvements
- **Documentation**: Guides, API docs, examples
- **Testing**: Test cases, bug reports
- **Design**: UI/UX improvements, accessibility
- **Community**: Help, support, evangelism

Thank you for contributing to Scriberr! Your contributions help make audio transcription more accessible and powerful for everyone. 