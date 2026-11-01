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
    background: #f4f1e8;
    color: #111;
    border: 2px solid #05040a;
    box-shadow: 2px 2px 0 #05040a;
  }
  .round {
    border-radius: 50%;
    padding: 0;
  }
  .xbox.south { background: #5dc44f; }
  .xbox.east { background: #c62828; color: #fff; }
  .xbox.west { background: #1d5fd6; color: #fff; }
  .xbox.north { background: #f5c518; }
  .ps.south { color: #3b82f6; background: #111; }
  .ps.east { color: #e5484d; background: #111; }
  .ps.west { color: #e879b8; background: #111; }
  .ps.north { color: #2fbf9a; background: #111; }
  :global(body.high-contrast) .glyph {
    background: #fff !important;
    color: #000 !important;
  }
</style>
