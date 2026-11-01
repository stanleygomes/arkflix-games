<script lang="ts">
  // Menu de opções da lista: ordenação (e, depois, filtros). Prende o foco
  // ([data-nav-scope]); voltar fecha.
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { SORTS } from '../lib/library'
  import { play } from '../lib/sound'

  let { sort, onsort, onclose }: { sort: string; onsort: (key: string) => void; onclose: () => void } = $props()

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back' || a === 'menu') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'search'
    })
  })
</script>

<div class="backdrop">
  <div class="menu panel" role="dialog" aria-modal="true" aria-labelledby="options-title" data-nav-scope>
    <h2 id="options-title">Opções da lista</h2>
    <fieldset>
      <legend>Ordenar por</legend>
      <div class="options" role="radiogroup" aria-label="Ordenar por">
        {#each SORTS as s (s.key)}
          <button
            class="option"
            role="radio"
            aria-checked={sort === s.key}
            data-nav
            data-nav-default={sort === s.key ? '' : undefined}
            onclick={() => onsort(s.key)}
          >
            <span class="dot" aria-hidden="true"></span>{s.label}
          </button>
        {/each}
      </div>
    </fieldset>
    <button class="btn" data-nav onclick={onclose}>Fechar</button>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    display: grid;
    place-items: center;
    z-index: 80;
  }
  .menu {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-width: 24rem;
    max-height: 90vh;
    overflow-y: auto;
  }
  fieldset {
    border: none;
    margin: 0;
    padding: 0;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.6rem;
  }
  .options {
    display: grid;
    gap: 0.3rem;
  }
  .option {
    font: inherit;
    color: var(--text);
    background: transparent;
    border: 3px solid transparent;
    text-align: left;
    padding: 0.45rem 0.6rem;
    display: flex;
    gap: 0.7rem;
    align-items: center;
    cursor: pointer;
  }
  .dot {
    width: 0.9rem;
    height: 0.9rem;
    border: 2px solid var(--text);
    flex: none;
  }
  .option[aria-checked='true'] .dot {
    background: var(--accent);
    box-shadow: inset 0 0 0 2px var(--surface);
  }
</style>
