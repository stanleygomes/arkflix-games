<script lang="ts">
  // Biblioteca em grade de uma fonte (console, coleção, lista), com ordenação.
  import GameGrid from '../components/GameGrid.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import ListOptions from '../components/ListOptions.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { focusedGameId, toggleFavorite } from '../lib/actions'
  import { pushHandler } from '../lib/input'
  import { SORTS, filterGames, filterGroups, pickRandom, sortGames } from '../lib/library'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { filtersOf, isSmart, sortOf, sourceGames, sourceTitle } from '../lib/sources'
  import { focusEl } from '../lib/nav'
  import { api } from '../lib/api'
  import { app, back, confirm, prompt, putList, system, toast } from '../lib/state.svelte'
  import type { Settings } from '../lib/types'
  import { saveSoon } from './settings/save.svelte'

  let { source }: { source: string } = $props()
  let options = $state(false)
  let lastRandom = ''
  const sort = $derived(sortOf(source))
  const filters = $derived(filtersOf(source))
  const all = $derived(sourceGames(source))
  const groups = $derived(filterGroups(all))
  const games = $derived(sortGames(filterGames(all, filters), sort))
  const title = $derived(sourceTitle(source))

  $effect(() => {
    const sys = source.startsWith('system:') ? system(source.slice(7)) : undefined
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : (sys?.palette ?? homePalette))
  })

  function saveUI(sortKey: string, active: string[]) {
    const s = app.settings as Settings
    saveSoon({
      ...s,
      ui: { sort: { ...(s.ui?.sort ?? {}), [source]: sortKey }, filters: { ...(s.ui?.filters ?? {}), [source]: active } },
    })
    play('toggle')
  }
  const setSort = (key: string) => saveUI(key, filters)
  const toggleFilter = (id: string) => saveUI(sort, filters.includes(id) ? filters.filter((f) => f !== id) : [...filters, id])
  const clearFilters = () => saveUI(sort, [])

  const listId = $derived(source.startsWith('list:') ? source.slice(5) : '')
  const smart = $derived(isSmart(app.lists.find((l) => l.id === listId)))
  const extra = $derived(
    listId
      ? [
          { label: 'Renomear lista', onclick: renameList },
          { label: 'Apagar lista', danger: true, onclick: deleteList },
        ]
      : filters.length
        ? [{ label: 'Salvar como lista automática', onclick: saveSmart }]
        : [],
  )

  // nome sugerido: a fonte e os filtros ("Super Nintendo: RPG, Nunca jogados")
  function suggestName(): string {
    const labels = filters.map((f) => groups.flatMap((g) => g.options).find((o) => o.id === f)?.label ?? f.split(':').pop())
    const name = `${title}: ${labels.join(', ')}`
    return name.length > 40 ? name.slice(0, 39).trimEnd() + '…' : name
  }

  async function saveSmart() {
    options = false
    await prompt('Nome da lista automática', suggestName(), 'Criar lista', async (name) => {
      try {
        const l = await api.createSmartList(name, { source, filters: [...filters], sort })
        putList(l)
        toast(`Lista automática "${l.name}" criada: está no Início e se atualiza sozinha`, 'ok', 4000)
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 40 })
  }

  async function renameList() {
    options = false
    await prompt('Renomear lista', title, 'Salvar', async (name) => {
      try {
        putList(await api.renameList(listId, name))
        return null
      } catch (err) {
        return (err as Error).message
      }
    })
  }

  async function deleteList() {
    options = false
    if (!(await confirm(`Apagar a lista "${title}"? Os jogos continuam na biblioteca.`, 'Apagar', true))) return
    try {
      await api.deleteList(listId)
      app.lists = app.lists.filter((l) => l.id !== listId)
      if (app.homeTab === source) app.homeTab = 'home'
      toast('Lista apagada', 'ok', 2500)
      back()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() =>
    pushHandler((a) => {
      if (a === 'fav') {
        toggleFavorite(focusedGameId())
        return true
      }
      if (a === 'random') {
        const pick = pickRandom(games, lastRandom)
        if (pick) {
          lastRandom = pick.id
          play('start')
          toast(`Surpresa: ${pick.name}`, 'info', 3000)
          focusEl(document.querySelector<HTMLElement>(`.grid [data-game="${pick.id}"]`))
        }
        return true
      }
      if (app.prompt || app.confirm) return false
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
      {#if smart}
        <span class="smart muted"><Icon name="auto" /> Lista automática</span>
      {/if}
      <span class="count muted">
        {games.length} jogo{games.length === 1 ? '' : 's'}{filters.length ? ` de ${all.length}` : ''}
      </span>
      {#if filters.length}
        <span class="chip" role="status">{filters.length} filtro{filters.length === 1 ? '' : 's'} ativo{filters.length === 1 ? '' : 's'}</span>
        <button class="btn small" data-nav onclick={clearFilters}>Limpar filtros</button>
      {/if}
      <button class="btn small" data-nav onclick={() => (options = true)} aria-haspopup="dialog">
        <Icon name="sliders" /> {SORTS.find((s) => s.key === sort)?.label ?? 'Nome (A–Z)'}
      </button>
    </header>
    {#if games.length}
      <GameGrid {games} label={`Jogos: ${title}`} showSystem={!source.startsWith('system:')} />
    {:else if filters.length}
      <p class="muted">Nenhum jogo com esses filtros. Tire algum filtro ou limpe todos.</p>
    {:else}
      <p class="muted">Nenhum jogo aqui.</p>
    {/if}
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Favorito' },
      { button: 'start', label: 'Opções' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

{#if options}
  <ListOptions {sort} onsort={setSort} onclose={() => (options = false)} {groups} {filters} onfilter={toggleFilter} onclear={clearFilters} {extra} />
{/if}

<style>
  .page {
    height: 100%;
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
  .smart {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.9rem;
  }
</style>
