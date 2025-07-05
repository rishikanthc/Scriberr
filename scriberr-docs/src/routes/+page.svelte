<script lang="ts">
  import { Card } from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Carousel, CarouselContent, CarouselItem, CarouselNext, CarouselPrevious } from '$lib/components/ui/carousel';
  import { Dialog, DialogContent, DialogTrigger } from '$lib/components/ui/dialog';
  import { LucideSparkles, LucideMic, LucideYoutube, LucideDownload, LucideUser, LucideMessageCircle, LucideSettings2, LucideZoomIn } from 'lucide-svelte';
  import Logo from '$lib/components/Logo.svelte';

  // State for zoom modal
  let selectedScreenshot: typeof screenshots[0] | null = null;
  let isDialogOpen = false;

  // Screenshots with captions
  const screenshots = [
    { src: '/homepage.png', alt: 'Homepage', caption: 'Clean and intuitive homepage interface' },
    { src: '/chat-with-transcript.png', alt: 'Chat with Transcript', caption: 'AI-powered chat with your transcript' },
    { src: '/audio-recorder-window.png', alt: 'Audio Recorder', caption: 'Built-in audio recorder for instant transcription' },
    { src: '/transcript-window.png', alt: 'Transcript Window', caption: 'Real-time transcript display with playback controls' },
    { src: '/transcript-window-diarization.png', alt: 'Diarization', caption: 'Speaker diarization with color-coded speakers' },
    { src: '/summarization.png', alt: 'Summarization', caption: 'Automatic summarization with customizable prompts' },
    { src: '/summary-markdown-preview.png', alt: 'Summary Markdown Preview', caption: 'Rich markdown preview for summaries' },
    { src: '/youtube-download.png', alt: 'YouTube Download', caption: 'Direct YouTube video transcription' },
    { src: '/transcription-settings.png', alt: 'Transcription Settings', caption: 'Advanced transcription model settings' },
    { src: '/audio-options.png', alt: 'Audio Options', caption: 'Flexible audio input options' },
    { src: '/right-click-context-menu.png', alt: 'Context Menu', caption: 'Right-click context menu for quick actions' },
    { src: '/active-jobs.png', alt: 'Active Jobs', caption: 'Monitor active transcription jobs' },
    { src: '/transcript-download-options.png', alt: 'Download Options', caption: 'Multiple export formats (TXT, JSON, SRT)' },
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

  function openZoom(screenshot: typeof screenshots[0]) {
    selectedScreenshot = screenshot;
    isDialogOpen = true;
  }
</script>

<section class="min-h-screen bg-background text-foreground flex flex-col items-center px-4 py-12">
  <div class="max-w-3xl w-full text-center mb-12">

    <h1 class="text-4xl md:text-5xl font-bold mb-4 tracking-tight flex items-center justify-center gap-3">
      <Logo size={64} strokeColor="#daff7d" />
      Scriberr
    </h1>
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
      <Button variant="outline" size="lg" href="/docs">
        Documentation
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
    <h2 class="text-2xl font-bold mb-8 text-center">Screenshots</h2>
    <Carousel class="w-full">
      <CarouselContent>
        {#each screenshots as shot}
          <CarouselItem class="md:basis-1/2 lg:basis-1/3">
            <div class="p-2">
              <Card class="overflow-hidden border border-border bg-card/70 shadow-lg">
                <div class="aspect-[4/3] overflow-hidden relative group cursor-pointer" on:click={() => openZoom(shot)}>
                  <img 
                    src={shot.src} 
                    alt={shot.alt} 
                    class="w-full h-full object-cover object-top hover:scale-105 transition-transform duration-300" 
                    loading="lazy" 
                  />
                  <div class="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity duration-200 flex items-center justify-center">
                    <LucideZoomIn class="w-8 h-8 text-white" />
                  </div>
                </div>
                <div class="p-4 pt-3">
                  <p class="text-sm text-muted-foreground text-center font-medium leading-relaxed">{shot.caption}</p>
                </div>
              </Card>
            </div>
          </CarouselItem>
        {/each}
      </CarouselContent>
      <CarouselPrevious />
      <CarouselNext />
    </Carousel>
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

<!-- Zoom Modal -->
<Dialog bind:open={isDialogOpen}>
  <DialogContent class="max-w-4xl w-[90vw] max-h-[90vh] p-0 overflow-hidden">
    {#if selectedScreenshot}
      <div class="relative">
        <img 
          src={selectedScreenshot.src} 
          alt={selectedScreenshot.alt} 
          class="w-full h-auto max-h-[80vh] object-contain" 
        />
        <div class="absolute bottom-0 left-0 right-0 bg-gradient-to-t from-black/80 to-transparent p-4">
          <p class="text-white text-center font-medium">{selectedScreenshot.caption}</p>
        </div>
      </div>
    {/if}
  </DialogContent>
</Dialog>
