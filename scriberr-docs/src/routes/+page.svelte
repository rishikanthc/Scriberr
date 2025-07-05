<script lang="ts">
  import { Card } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { LucideSparkles, LucideMic, LucideYoutube, LucideDownload, LucideUser, LucideMessageCircle, LucideSettings2 } from 'lucide-svelte';

  // Screenshots
  const screenshots = [
    { src: '/homepage.png', alt: 'Homepage' },
    { src: '/chat-with-transcript.png', alt: 'Chat with Transcript' },
    { src: '/audio-recorder-window.png', alt: 'Audio Recorder' },
    { src: '/transcript-window.png', alt: 'Transcript Window' },
    { src: '/transcript-window-diarization.png', alt: 'Diarization' },
    { src: '/summarization.png', alt: 'Summarization' },
    { src: '/summary-markdown-preview.png', alt: 'Summary Markdown Preview' },
    { src: '/youtube-download.png', alt: 'YouTube Download' },
    { src: '/transcription-settings.png', alt: 'Transcription Settings' },
    { src: '/audio-options.png', alt: 'Audio Options' },
    { src: '/right-click-context-menu.png', alt: 'Context Menu' },
    { src: '/active-jobs.png', alt: 'Active Jobs' },
    { src: '/transcript-download-options.png', alt: 'Download Options' },
  ];

  // Features
  const features = [
    {
      icon: LucideMic,
      title: 'Fast Transcription',
      desc: 'Transcribe audio quickly with support for all model sizes and automatic language detection.'
    },
    {
      icon: LucideUser,
      title: 'Speaker Diarization',
      desc: 'Detect and identify speakers in your audio with advanced diarization.'
    },
    {
      icon: LucideSparkles,
      title: 'Automatic Summarization',
      desc: 'Generate summaries using OpenAI or Ollama endpoints.'
    },
    {
      icon: LucideMessageCircle,
      title: 'AI Chat with Notes',
      desc: 'Chat with your transcript and take notes, powered by OpenAI/Ollama.'
    },
    {
      icon: LucideMic,
      title: 'Built-in Audio Recorder',
      desc: 'Record audio directly in the app for instant transcription.'
    },
    {
      icon: LucideYoutube,
      title: 'YouTube Audio Transcription',
      desc: 'Transcribe audio from YouTube videos with ease.'
    },
    {
      icon: LucideDownload,
      title: 'Flexible Export',
      desc: 'Download transcripts as plaintext, JSON, or SRT files.'
    },
    {
      icon: LucideSettings2,
      title: 'Advanced Controls',
      desc: 'Tweak advanced parameters for transcription and diarization models.'
    }
  ];
</script>

<section class="min-h-screen bg-background text-foreground flex flex-col items-center px-4 py-12">
  <div class="max-w-3xl w-full text-center mb-12">
    <img src="/favicon.svg" alt="Scriberr Logo" class="mx-auto mb-6 w-16 h-16" />
    <h1 class="text-4xl md:text-5xl font-bold mb-4 tracking-tight">Scriberr</h1>
    <p class="text-lg md:text-xl text-muted-foreground mb-4">
      Self-hosted, fully local audio transcription app. Fast, private, and feature-rich.
    </p>
    <p class="text-md text-muted-foreground mb-2">
      <span class="font-semibold">v1.0.0-beta1</span> &mdash; Public beta available now!
    </p>
    <div class="flex justify-center gap-4 mt-6">
      <Button size="lg" href="https://github.com/noeticgeek/scriberr" target="_blank" rel="noopener">
        View on GitHub
      </Button>
      <Button variant="secondary" size="lg" href="#features">
        See Features
      </Button>
    </div>
  </div>

  <div id="features" class="max-w-5xl w-full grid grid-cols-1 md:grid-cols-2 gap-6 mb-16">
    {#each features as feature}
      <Card class="flex flex-row items-center gap-4 p-6 bg-card/80 border border-border shadow-md">
        <svelte:component this={feature.icon} class="w-10 h-10 text-primary shrink-0" />
        <div class="text-left">
          <h3 class="text-xl font-semibold mb-1">{feature.title}</h3>
          <p class="text-muted-foreground text-sm">{feature.desc}</p>
        </div>
      </Card>
    {/each}
  </div>

  <div class="max-w-6xl w-full mb-20">
    <h2 class="text-2xl font-bold mb-6 text-center">Screenshots</h2>
    <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
      {#each screenshots as shot}
        <div class="rounded-lg overflow-hidden border border-border bg-card/70 shadow">
          <img src={shot.src} alt={shot.alt} class="w-full h-32 object-cover object-top md:h-40 hover:scale-105 transition-transform duration-200" loading="lazy" />
        </div>
      {/each}
    </div>
  </div>

  <div class="max-w-2xl w-full text-center mt-8">
    <Card class="p-8 bg-primary text-primary-foreground border-none shadow-lg">
      <h2 class="text-2xl font-bold mb-2">Ready to try Scriberr?</h2>
      <p class="mb-6 text-lg">Download the latest release or check out the docs to get started with your own private, local transcription server.</p>
      <div class="flex justify-center gap-4">
        <Button size="lg" href="https://github.com/noeticgeek/scriberr/releases" target="_blank" rel="noopener">
          Download
        </Button>
        <Button variant="secondary" size="lg" href="https://github.com/noeticgeek/scriberr#readme" target="_blank" rel="noopener">
          Read the Docs
        </Button>
      </div>
    </Card>
  </div>
</section>
