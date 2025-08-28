<script lang="ts">
  import { locale } from 'svelte-i18n';
  import { browser } from '$app/environment';

  const locales = {
    'en': 'English',
    'zh-TW': '繁體中文'
  };

  function handleChange(event: Event) {
    const newLocale = (event.target as HTMLSelectElement).value;
    if (newLocale) {
      locale.set(newLocale);
      if (browser) {
        localStorage.setItem('selectedLocale', newLocale);
      }
    }
  }

  // Set the initial value of the select dropdown from localStorage if available
  let selectedLocale = 'en';
  if (browser) {
    selectedLocale = localStorage.getItem('selectedLocale') || $locale || 'en';
  }
</script>

<div class="language-switcher">
  <select bind:value={selectedLocale} on:change={handleChange} aria-label="Language selector">
    {#each Object.entries(locales) as [value, name]}
      <option {value}>{name}</option>
    {/each}
  </select>
</div>

<style>
  .language-switcher {
    position: relative;
    display: inline-block;
  }
  select {
    padding: 8px 12px;
    border-radius: 4px;
    border: 1px solid #ccc;
    background-color: white;
    cursor: pointer;
  }
</style>
