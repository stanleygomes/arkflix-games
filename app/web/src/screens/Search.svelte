<script lang="ts">
  import GameGrid from '../components/GameGrid.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Keyboard from '../components/Keyboard.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { focusedGameId, toggleFavorite } from '../lib/actions'
  import { pushHandler } from '../lib/input'
  import { search } from '../lib/library'
  import { focusEl } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app } from '../lib/state.svelte'

  let query = $state('')
  const results = $derived(search(app.games, query))
  // quem usa teclado físico começa no campo; quem usa controle, no teclado na tela
  const typing = app.family === 'keyboard'

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
  })

  // F/Y favorita o resultado em foco (nos campos de texto o F digita)
  $effect(() =>
    pushHandler((a) => {
      if (a === 'fav' && focusedGameId()) {
        toggleFavorite(focusedGameId())
        return true
      }
      return false
    }),
  )

  function toResults() {
    focusEl(document.querySelector<HTMLElement>('.results [data-game]'))
  }
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="layout">
    <section class="query" aria-labelledby="search-title">
      <h1 class="pixel" id="search-title">Buscar</h1>
      <label class="sr-only" for="search-input">Nome do jogo</label>
      <input
        id="search-input"
        type="text"
        data-nav
        data-nav-default={typing ? '' : undefined}
        bind:value={query}
        placeholder="Nome do jogo"
        autocomplete="off"
        spellcheck="false"
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            // sem isso o mesmo Enter "clica" no resultado que acabou de receber o foco
            e.preventDefault()
            toResults()
          }
        }}
      />
      <Keyboard bind:value={query} onsubmit={toResults} submitLabel="Ver resultados" autofocus={!typing} />
    </section>
    <section class="results" aria-live="polite" aria-labelledby="results-count">
      <p id="results-count" class="count">
        {#if !query.trim()}
          Digite parte do nome. Acento e maiúscula não importam.
        {:else if results.length === 0}
          Nada encontrado pra "{query}".
        {:else}
          {results.length} jogo{results.length === 1 ? '' : 's'}
        {/if}
      </p>
      {#if results.length}
        <GameGrid games={results} label="Resultados da busca" />
      {/if}
    </section>
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Digitar / abrir' }, { button: 'east', label: 'Voltar' }]} />
</div>

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 2.5rem;
    padding: 0.5rem 2.5rem 1rem 3.5rem;
  }
  .query {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h1 {
    font-size: 2.2rem;
  }
  .results {
    overflow-y: auto;
    min-height: 0;
  }
  .count {
    color: var(--muted);
    margin-bottom: 0.6rem;
  }
</style>
