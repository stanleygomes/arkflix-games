<script lang="ts">
  // Símbolo do botão no estilo do controle em uso (A/B do Xbox, ✕/○ do PS...).
  import { glyph } from '../lib/keys'
  import type { Family } from '../lib/types'

  let { button, family }: { button: string; family: Family } = $props()
  const text = $derived(glyph(button, family))
  const ps = $derived(family === 'playstation' && ['south', 'east', 'west', 'north'].includes(button))
</script>

<kbd class="glyph {family} {button}" class:round={['south', 'east', 'west', 'north'].includes(button) && family !== 'keyboard'} class:ps>{text}</kbd>

<style>
  .glyph {
    font-family: var(--font-title);
    font-weight: 700;
    font-size: 0.8rem;
    min-width: 1.7em;
    height: 1.7em;
    padding: 0 0.35em;
    display: inline-grid;
    place-items: center;
    background: #000;
    color: var(--accent-2);
    border: 2px solid currentColor;
    box-shadow: 0 0 6px color-mix(in srgb, currentColor 60%, transparent);
  }
  .round {
    border-radius: 50%;
    padding: 0;
  }
  .xbox.south { color: #5dff4f; }
  .xbox.east { color: #ff5a6a; }
  .xbox.west { color: #5c9dff; }
  .xbox.north { color: #ffe600; }
  .ps.south { color: #3b82f6; background: #111; }
  .ps.east { color: #e5484d; background: #111; }
  .ps.west { color: #e879b8; background: #111; }
  .ps.north { color: #2fbf9a; background: #111; }
  :global(body.high-contrast) .glyph {
    background: #fff !important;
    color: #000 !important;
  }
</style>
