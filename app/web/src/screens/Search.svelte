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
  import { app, go, isKids } from '../lib/state.svelte'
  import { shortcuts, type Shortcut } from '../lib/shortcuts'
  import Icon from '../components/Icon.svelte'

  let query = $state('')
  const results = $derived(search(app.games, query))
  const extra = $derived(
    shortcuts(query, {
      systems: app.systems,
      withGames: new Set(app.games.map((g) => g.system)),
      lists: app.lists,
      kids: isKids(),
      desktop: app.mode === 'desktop',
    }),
  )
  const icons = { console: 'pad', list: 'star', setting: 'gear' } as const

  function openShortcut(s: Shortcut) {
    if (s.power) app.power = true
    else if (s.go) go(s.go)
  }
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
    // jogo primeiro (é o que mais se busca); sem jogo, o primeiro atalho
    focusEl(document.querySelector<HTMLElement>('.results [data-game]') ?? document.querySelector<HTMLElement>('.results [data-nav]'))
  }
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="layout">
    <section class="query" aria-labelledby="search-title">
      <h1 class="pixel" id="search-title">Buscar</h1>
      <label class="sr-only" for="search-input">Buscar jogo, console ou configuração</label>
      <input
        id="search-input"
        type="text"
        data-nav
        data-nav-default={typing ? '' : undefined}
        bind:value={query}
        placeholder="Jogo, console ou configuração"
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
          Jogo, console, lista ou configuração ("bluetooth", "tela cheia"). Acento e maiúscula não importam.
        {:else if results.length === 0 && extra.length === 0}
          Nada encontrado pra "{query}".
        {:else}
          {results.length} jogo{results.length === 1 ? '' : 's'}{extra.length ? ` e ${extra.length} atalho${extra.length === 1 ? '' : 's'}` : ''}
        {/if}
      </p>
      {#if extra.length}
        <ul class="shortcuts" aria-label="Consoles, listas e configurações">
          {#each extra as s (s.id)}
            <li>
              <button class="shortcut" data-nav data-shortcut={s.kind} onclick={() => openShortcut(s)}>
                <Icon name={icons[s.kind]} />
                <span class="label">{s.label}</span>
                <span class="hint muted">{s.hint}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
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
  .shortcuts {
    list-style: none;
    margin: 0 0 1rem;
    padding: 0.3rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
  }
  .shortcut {
    display: grid;
    grid-template-columns: auto auto;
    gap: 0.1rem 0.6rem;
    align-items: center;
    padding: 0.6rem 0.9rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--surface) 85%, transparent);
    border: 2px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
    cursor: pointer;
  }
  .shortcut :global(svg) {
    grid-row: span 2;
    color: var(--accent);
  }
  .shortcut .label {
    font-weight: 700;
  }
  .shortcut .hint {
    font-size: 0.8rem;
  }
  .count {
    color: var(--muted);
    margin-bottom: 0.6rem;
  }
</style>
