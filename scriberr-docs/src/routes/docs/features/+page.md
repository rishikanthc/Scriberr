---
layout: _docs
---

# Features

Scriberr offers a comprehensive set of features for audio transcription and analysis. This guide covers all the capabilities available in the current version.

## Core Transcription Features

### Fast Audio Transcription

Scriberr provides lightning-fast transcription using OpenAI's Whisper models with advanced optimizations:

- **Multiple Model Sizes**: Choose from tiny, base, small, medium, or large models
- **Automatic Language Detection**: Supports 99+ languages with automatic detection
- **High Accuracy**: State-of-the-art speech recognition with WhisperX alignment
- **Batch Processing**: Transcribe multiple files simultaneously
- **Real-time Progress**: Live updates during transcription

### Speaker Diarization

Automatically identify and separate different speakers in your audio:

- **Speaker Detection**: Identify when different people are speaking
- **Color-coded Output**: Each speaker gets a unique color for easy identification
- **Speaker Labels**: Automatic labeling (Speaker 1, Speaker 2, etc.)
- **Timeline Visualization**: See speaker changes over time
- **Export Support**: Maintain speaker information in exported files

### Advanced Audio Processing

Enhanced audio processing for better transcription quality:

- **Voice Activity Detection (VAD)**: Remove silence and background noise
- **Automatic Speech Recognition (ASR)**: Optimized for speech clarity
- **Audio Alignment**: Precise timing alignment with WhisperX
- **Multiple Audio Formats**: Support for MP3, WAV, M4A, FLAC, and more
- **Audio Normalization**: Automatic volume and quality optimization

## AI-Powered Features

### Automatic Summarization

Generate intelligent summaries of your transcripts:

- **Custom Prompts**: Create and save custom summarization templates
- **Multiple Styles**: Academic, business, casual, or custom summaries
- **Markdown Support**: Rich formatting with markdown preview
- **AI Integration**: Works with OpenAI API or local Ollama models
- **Export Options**: Save summaries in various formats

### AI Chat with Transcripts

Interact with your transcripts using AI:

- **Contextual Chat**: Ask questions about your transcript content
- **Multiple Sessions**: Create separate chat sessions for different topics
- **Note Taking**: Save important insights and notes
- **Search Integration**: Find specific content quickly
- **Export Conversations**: Save chat history and insights

### Template Management

Create and manage reusable prompt templates:

- **Custom Templates**: Build templates for different use cases
- **Template Library**: Save and organize your templates
- **Quick Access**: Apply templates with one click
- **Sharing**: Export and import templates
- **Categories**: Organize templates by purpose

## Audio Input Options

### Built-in Audio Recorder

Record audio directly within the application:

- **High-Quality Recording**: Professional-grade audio capture
- **Real-time Monitoring**: See audio levels while recording
- **Format Options**: Choose from multiple output formats
- **Device Selection**: Select input devices and microphones
- **Recording Controls**: Pause, resume, and stop functionality

### File Upload

Upload existing audio files for transcription:

- **Drag & Drop**: Simple file upload interface
- **Multiple Files**: Upload several files at once
- **Format Support**: Wide range of audio formats
- **File Validation**: Automatic format and size checking
- **Progress Tracking**: Upload progress indicators

### YouTube Integration

Transcribe audio from YouTube videos:

- **Direct URL Input**: Paste YouTube URLs for transcription
- **Audio Extraction**: Automatically extract audio from videos
- **Quality Selection**: Choose audio quality for download
- **Batch Processing**: Process multiple YouTube videos
- **Metadata Preservation**: Keep video information

## Export and Output Options

### Multiple Export Formats

Export your transcripts in various formats:

- **Plain Text (.txt)**: Simple text format
- **JSON (.json)**: Structured data with metadata
- **SRT (.srt)**: Subtitle format for video editing
- **Markdown (.md)**: Rich text with formatting
- **Custom Formats**: Extensible export system

### Advanced Export Features

Enhanced export capabilities:

- **Speaker Information**: Include speaker diarization data
- **Timestamps**: Precise timing information
- **Metadata**: File information and processing details
- **Batch Export**: Export multiple files at once
- **Custom Styling**: Format exports to your preferences

## User Interface Features

### Modern Web Interface

Clean, responsive design built with SvelteKit:

- **Dark Mode**: Eye-friendly dark theme
- **Responsive Design**: Works on desktop, tablet, and mobile
- **Keyboard Shortcuts**: Power user shortcuts for efficiency
- **Accessibility**: WCAG compliant interface
- **Customizable**: Theme and layout options

### Audio Playback Integration

Advanced audio playback with transcript synchronization:

- **Synchronized Playback**: Audio and transcript stay in sync
- **Click to Jump**: Click any transcript segment to jump to that time
- **Playback Controls**: Standard audio controls with transcript highlighting
- **Speed Control**: Adjust playback speed
- **Loop Sections**: Repeat specific segments

### Job Management

Monitor and manage transcription jobs:

- **Job Queue**: View all active and completed jobs
- **Progress Tracking**: Real-time progress updates
- **Job History**: Complete history of all transcriptions
- **Error Handling**: Clear error messages and retry options
- **Resource Monitoring**: CPU and memory usage

## Advanced Configuration

### Model Settings

Fine-tune transcription models:

- **Model Selection**: Choose optimal model for your needs
- **Device Configuration**: CPU, CUDA, or MPS acceleration
- **Parameter Tuning**: Adjust model parameters for accuracy vs speed
- **Custom Models**: Load custom trained models
- **Performance Optimization**: Balance speed and accuracy

### System Configuration

Advanced system settings:

- **Storage Management**: Configure data and audio directories
- **Memory Limits**: Set memory usage limits
- **Concurrent Jobs**: Control number of simultaneous transcriptions
- **Network Settings**: Proxy and firewall configuration
- **Logging**: Detailed logging for debugging

## Privacy and Security

### Complete Local Processing

Your data never leaves your machine:

- **No Cloud Dependencies**: All processing happens locally
- **Data Privacy**: Audio files and transcripts stay on your system
- **Offline Operation**: Works without internet connection
- **Audit Trail**: Complete transparency of data handling
- **Compliance Ready**: Meets strict privacy requirements

### Security Features

Enterprise-grade security:

- **Authentication**: Optional user authentication
- **Access Control**: Role-based permissions
- **Data Encryption**: Optional encryption for stored data
- **Secure Communication**: HTTPS support
- **Audit Logging**: Security event logging

## Performance and Scalability

### High Performance

Optimized for speed and efficiency:

- **Go Backend**: High-performance Go server
- **Efficient Processing**: Optimized audio processing pipeline
- **Memory Management**: Intelligent memory usage
- **Caching**: Model and result caching
- **Parallel Processing**: Multi-threaded operations

### Scalability

Designed to handle growing workloads:

- **Horizontal Scaling**: Run multiple instances
- **Load Balancing**: Distribute work across instances
- **Resource Scaling**: Automatic resource allocation
- **Queue Management**: Intelligent job queuing
- **Performance Monitoring**: Built-in performance metrics

## Integration Capabilities

### API Access

Programmatic access to Scriberr:

- **REST API**: Full REST API for integration
- **WebSocket Support**: Real-time communication
- **Authentication**: API key authentication
- **Rate Limiting**: Configurable rate limits
- **Documentation**: Complete API documentation

### Third-party Integrations

Connect with other tools and services:

- **Webhook Support**: Notify external systems
- **Export Integrations**: Connect to cloud storage
- **Workflow Automation**: Integrate with automation tools
- **Custom Extensions**: Plugin system for custom functionality
- **Standard Formats**: Support for industry standards 