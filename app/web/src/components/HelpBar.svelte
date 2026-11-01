<script lang="ts">
  // Dicas de botão no rodapé, com os símbolos do controle em uso.
  import { glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  let { hints }: { hints: { button: string; label: string }[] } = $props()
  const family = $derived(glyphFamily())
</script>

<footer class="help" aria-label="Atalhos">
  {#each hints as h (h.button + h.label)}
    <span class="hint"><Glyph button={h.button} {family} /> {h.label}</span>
  {/each}
</footer>

<style>
  .help {
    display: flex;
    gap: 1.6rem;
    justify-content: flex-end;
    padding: 0.8rem 2.5rem 1.2rem;
    color: var(--muted);
    font-size: 0.95rem;
  }
  .hint {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
  }
</style>
