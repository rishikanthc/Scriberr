<script lang="ts">
  import { onMount } from 'svelte';
  import WhisperSetup from './WhisperSetup.svelte';
  import { Button } from '$lib/components/ui/button/index.js';
  
  // Configuration state
  let isSetupComplete = $state(false);
  let step = $state(1);
  
  function handleSetupComplete(event) {
    isSetupComplete = event.detail.complete;
    if (isSetupComplete) {
      step = 2;
    }
  }
  
  function reloadApp() {
    window.location.href = '/';
  }
</script>

<div class="p-4 space-y-8">
  <h2 class="text-2xl font-semibold">System Configuration</h2>
  
  <div class="space-y-2">
    <div class="flex items-center space-x-2">
      <div class="w-8 h-8 rounded-full bg-primary text-white flex items-center justify-center">
        {step > 1 ? '✓' : '1'}
      </div>
      <div class="font-medium">Speech Recognition Setup</div>
    </div>
    
    {#if step === 1}
      <div class="pl-10">
        <WhisperSetup on:setupcomplete={handleSetupComplete} />
      </div>
    {/if}
    
    {#if step === 2}
      <div class="pl-10 py-4">
        <div class="bg-green-50 p-4 rounded border border-green-200">
          <p class="text-green-800">Setup completed successfully!</p>
          <Button class="mt-4" on:click={reloadApp}>
            Go to Application
          </Button>
        </div>
      </div>
    {/if}
  </div>
</div>