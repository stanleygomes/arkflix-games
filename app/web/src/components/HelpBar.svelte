<script lang="ts">
  // Dicas de botão no rodapé, com os símbolos do controle em uso.
  import { glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  let { hints, coin = false }: { hints: { button: string; label: string }[]; coin?: boolean } = $props()
  const family = $derived(glyphFamily())
</script>

<footer class="help" aria-label="Atalhos">
  {#if coin}<span class="coin pixel" aria-hidden="true">INSERT COIN</span>{/if}
  {#each hints as h (h.button + h.label)}
    <span class="hint"><Glyph button={h.button} {family} /> {h.label}</span>
  {/each}
</footer>

<style>
  .help {
    display: flex;
    gap: 1.1rem;
    justify-content: flex-end;
    align-items: center;
    white-space: nowrap;
    padding: 0.8rem 2.5rem 1.2rem;
    color: var(--muted);
    font-size: 0.95rem;
  }
  .coin {
    margin-right: auto;
    color: var(--focus);
    font-size: 0.85rem;
    text-shadow: 0 0 10px var(--focus);
    animation: coin 1.1s steps(2, start) infinite;
  }
  @keyframes coin {
    to {
      visibility: hidden;
    }
  }
  .hint {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
  }
</style>
