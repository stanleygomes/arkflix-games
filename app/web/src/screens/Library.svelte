<script lang="ts">
  // Biblioteca em grade de uma fonte (console, coleção, lista), com ordenação.
  import GameGrid from '../components/GameGrid.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import ListOptions from '../components/ListOptions.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { pushHandler } from '../lib/input'
  import { SORTS, sortGames } from '../lib/library'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { sortOf, sourceGames, sourceTitle } from '../lib/sources'
  import { app, system } from '../lib/state.svelte'
  import type { Settings } from '../lib/types'
  import { saveSoon } from './settings/save.svelte'

  let { source }: { source: string } = $props()
  let options = $state(false)
  const sort = $derived(sortOf(source))
  const games = $derived(sortGames(sourceGames(source), sort))
  const title = $derived(sourceTitle(source))

  $effect(() => {
    const sys = source.startsWith('system:') ? system(source.slice(7)) : undefined
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : (sys?.palette ?? homePalette))
  })

  function setSort(key: string) {
    const s = app.settings as Settings
    saveSoon({ ...s, ui: { sort: { ...(s.ui?.sort ?? {}), [source]: key }, filters: s.ui?.filters ?? {} } })
    play('toggle')
  }

  $effect(() =>
    pushHandler((a) => {
      if (a === 'menu' && !options) {
        options = true
        play('accept')
        return true
      }
      return false
    }),
  )
</script>

<div class="page">
  <TopBar />
  <main class="body">
    <header class="head">
      <h1 class="pixel">{title}</h1>
      <span class="count muted">{games.length} jogo{games.length === 1 ? '' : 's'}</span>
      <button class="btn small" data-nav onclick={() => (options = true)} aria-haspopup="dialog">
        <Icon name="sliders" /> {SORTS.find((s) => s.key === sort)?.label ?? 'Nome (A–Z)'}
      </button>
    </header>
    {#if games.length}
      <GameGrid {games} label={`Jogos: ${title}`} showSystem={!source.startsWith('system:')} />
    {:else}
      <p class="muted">Nenhum jogo aqui.</p>
    {/if}
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Abrir' }, { button: 'start', label: 'Opções' }, { button: 'east', label: 'Voltar' }]} />
</div>

{#if options}
  <ListOptions {sort} onsort={setSort} onclose={() => (options = false)} />
{/if}

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 1rem 3.5rem;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 1.2rem;
    margin-bottom: 1rem;
    flex-wrap: wrap;
  }
  h1 {
    font-size: 2.2rem;
  }
</style>
