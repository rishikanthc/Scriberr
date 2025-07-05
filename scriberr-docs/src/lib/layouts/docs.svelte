<script lang="ts">
  import { page } from '$app/stores';
  import { Button } from '$lib/components/ui/button';
  import { LucideArrowLeft, LucideMenu, LucideX } from 'lucide-svelte';
  import Logo from '$lib/components/Logo.svelte';
  
  let mobileMenuOpen = false;

  const navigation = [
    { href: '/docs', label: 'Introduction' },
    { href: '/docs/installation', label: 'Installation' },
    { href: '/docs/features', label: 'Features' },
    { href: '/docs/usage', label: 'Usage' },
    { href: '/docs/compiling', label: 'Compiling from Source' },
    { href: '/docs/contributing', label: 'Contributing' }
  ];

  function toggleMobileMenu() {
    mobileMenuOpen = !mobileMenuOpen;
  }
</script>

<div class="min-h-screen bg-background text-foreground">
  <!-- Header -->
  <header class="border-b border-border bg-card/50 backdrop-blur-sm sticky top-0 z-50">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex justify-between items-center h-16">
        <div class="flex items-center space-x-4">
          <Button variant="ghost" size="sm" href="/" class="flex items-center space-x-2">
            <LucideArrowLeft class="w-4 h-4" />
            <span>Back to Home</span>
          </Button>
          <div class="hidden md:block w-px h-6 bg-border"></div>
          <h1 class="text-lg font-semibold flex items-center gap-2">
            <Logo size={24} strokeColor="#f0f9ff" />
            Scriberr Documentation
          </h1>
        </div>
        
        <!-- Mobile menu button -->
        <button 
          type="button"
          class="md:hidden inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background hover:bg-accent hover:text-accent-foreground h-9 px-3"
          on:click={toggleMobileMenu}
        >
          {#if mobileMenuOpen}
            <LucideX class="w-5 h-5" />
          {:else}
            <LucideMenu class="w-5 h-5" />
          {/if}
        </button>
      </div>
    </div>
  </header>

  <div class="flex">
    <!-- Sidebar -->
    <aside class="hidden md:block w-64 border-r border-border bg-card/30 min-h-screen sticky top-16">
      <nav class="p-6">
        <ul class="space-y-2">
          {#each navigation as item}
            <li>
              <a 
                href={item.href}
                class="block px-3 py-2 rounded-md text-sm transition-colors {$page.url.pathname === item.href 
                  ? 'bg-primary text-primary-foreground' 
                  : 'text-muted-foreground hover:text-foreground hover:bg-accent'}"
              >
                {item.label}
              </a>
            </li>
          {/each}
        </ul>
      </nav>
    </aside>

    <!-- Mobile menu -->
    {#if mobileMenuOpen}
      <div class="md:hidden fixed inset-0 z-40 bg-background/80 backdrop-blur-sm">
        <div class="fixed inset-y-0 left-0 w-64 bg-card border-r border-border p-6">
          <div class="flex justify-between items-center mb-6">
            <h2 class="text-lg font-semibold">Navigation</h2>
            <button 
              type="button"
              class="inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:opacity-50 disabled:pointer-events-none ring-offset-background hover:bg-accent hover:text-accent-foreground h-9 px-3"
              on:click={toggleMobileMenu}
            >
              <LucideX class="w-5 h-5" />
            </button>
          </div>
          <nav>
            <ul class="space-y-2">
              {#each navigation as item}
                <li>
                  <a 
                    href={item.href}
                    class="block px-3 py-2 rounded-md text-sm transition-colors {$page.url.pathname === item.href 
                      ? 'bg-primary text-primary-foreground' 
                      : 'text-muted-foreground hover:text-foreground hover:bg-accent'}"
                    on:click={() => mobileMenuOpen = false}
                  >
                    {item.label}
                  </a>
                </li>
              {/each}
            </ul>
          </nav>
        </div>
      </div>
    {/if}

    <!-- Main content -->
    <main class="flex-1 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <article class="prose prose-slate dark:prose-invert max-w-none">
        <slot />
      </article>
    </main>
  </div>
</div>

<style>
  :global(.prose) {
    --tw-prose-body: theme(colors.muted.foreground);
    --tw-prose-headings: theme(colors.foreground);
    --tw-prose-links: theme(colors.primary);
    --tw-prose-bold: theme(colors.foreground);
    --tw-prose-counters: theme(colors.muted.foreground);
    --tw-prose-bullets: theme(colors.muted.foreground);
    --tw-prose-hr: theme(colors.border);
    --tw-prose-quotes: theme(colors.muted.foreground);
    --tw-prose-quote-borders: theme(colors.border);
    --tw-prose-captions: theme(colors.muted.foreground);
    --tw-prose-code: theme(colors.foreground);
    --tw-prose-pre-code: theme(colors.muted.foreground);
    --tw-prose-pre-bg: theme(colors.card);
    --tw-prose-th-borders: theme(colors.border);
    --tw-prose-td-borders: theme(colors.border);
  }

  :global(.prose h1) {
    @apply text-3xl font-bold mb-8 pb-4 border-b border-border;
  }

  :global(.prose h2) {
    @apply text-2xl font-semibold mt-12 mb-6;
  }

  :global(.prose h3) {
    @apply text-xl font-semibold mt-8 mb-4;
  }

  :global(.prose p) {
    @apply mb-6 leading-relaxed;
  }

  :global(.prose ul) {
    @apply mb-6 space-y-2;
  }

  :global(.prose ol) {
    @apply mb-6 space-y-2;
  }

  :global(.prose li) {
    @apply leading-relaxed;
  }

  :global(.prose code) {
    @apply bg-muted px-1.5 py-0.5 rounded text-sm font-mono;
  }

  :global(.prose pre) {
    @apply bg-card border border-border rounded-lg p-4 overflow-x-auto;
  }

  :global(.prose pre code) {
    @apply bg-transparent p-0;
  }

  :global(.prose blockquote) {
    @apply border-l-4 border-primary bg-muted/50 pl-4 py-2 italic;
  }

  :global(.prose table) {
    @apply w-full border-collapse;
  }

  :global(.prose th) {
    @apply border border-border bg-muted px-4 py-2 text-left font-semibold;
  }

  :global(.prose td) {
    @apply border border-border px-4 py-2;
  }
</style> 