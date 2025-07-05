<script lang="ts">
  interface Props {
    size?: number;
    class?: string;
    strokeColor?: string;
  }

  let { size = 24, class: className = '', strokeColor = 'currentColor' } = $props();
</script>

<svg 
  width={size}
  height={size}
  viewBox="0 0 200 200" 
  fill="none" 
  xmlns="http://www.w3.org/2000/svg"
  class={className}
>
  <g transform="translate(10, 0) scale(0.85)">
    <!-- Wavy line -->
    <path 
      d="M20 120 C40 60, 60 60, 80 120 S120 180, 140 120"
      stroke={strokeColor} 
      stroke-width="8" 
      fill="none" 
      stroke-linecap="round"
    />

    <!-- Vertical bars -->
    <line x1="150" y1="100" x2="150" y2="130" stroke={strokeColor} stroke-width="8" stroke-linecap="round"/>
    <line x1="160" y1="90"  x2="160" y2="140" stroke={strokeColor} stroke-width="8" stroke-linecap="round"/>
    <line x1="170" y1="80"  x2="170" y2="150" stroke={strokeColor} stroke-width="8" stroke-linecap="round"/>
    <line x1="180" y1="90"  x2="180" y2="140" stroke={strokeColor} stroke-width="8" stroke-linecap="round"/>
    <line x1="190" y1="100" x2="190" y2="130" stroke={strokeColor} stroke-width="8" stroke-linecap="round"/>
  </g>
</svg> 