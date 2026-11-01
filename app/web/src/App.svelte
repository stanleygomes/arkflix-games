<script lang="ts">
  import { onMount, tick } from 'svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import Playing from './components/Playing.svelte'
  import PromptDialog from './components/PromptDialog.svelte'
  import Tasks from './components/Tasks.svelte'
  import Toasts from './components/Toasts.svelte'
  import { api, on } from './lib/api'
  import { startInput } from './lib/input'
  import { focusDefault, setReducedMotion } from './lib/nav'
  import { configure, pauseMusic, play } from './lib/sound'
  import { app, screen, toast } from './lib/state.svelte'
  import type { Game, GameEvent, Pad } from './lib/types'
  import Game_ from './screens/Game.svelte'
  import Home from './screens/Home.svelte'
  import Import from './screens/Import.svelte'
  import Search from './screens/Search.svelte'
  import Library from './screens/Library.svelte'
  import MetaEditor from './screens/MetaEditor.svelte'
  import Settings from './screens/Settings.svelte'

  const current = $derived(screen())

  onMount(async () => {
    try {
      const boot = await api.bootstrap()
      Object.assign(app, {
        version: boot.version,
        mode: boot.mode,
        systems: boot.systems,
        games: boot.games,
        settings: boot.settings,
        input: boot.input,
        retroPad: boot.retroPad,
        pads: boot.pads ?? [],
        lists: boot.lists ?? [],
        ready: true,
      })
      if (boot.pads?.length) app.family = boot.pads[0].family
      if (boot.running) app.playing = { state: 'running', id: boot.running, name: '' }
    } catch (err) {
      app.error = (err as Error).message
    }
    on<Pad[]>('pads', (pads) => {
      const before = app.pads.length
      app.pads = pads ?? []
      if (app.pads.length > before) toast(`Controle conectado: ${app.pads[app.pads.length - 1].name}`, 'ok', 3000)
      else if (app.pads.length < before) toast('Controle desconectado', 'info', 3000)
    })
    on<Game[]>('library', (games) => (app.games = games ?? []))
    on<GameEvent>('game', async (e) => {
      if (e.state === 'running') {
        app.playing = e
        pauseMusic(true)
        return
      }
      app.playing = null
      pauseMusic(false)
      if (e.error) {
        play('error')
        toast(`${e.name}: ${e.error}`, 'error', 9000)
      } else {
        const min = Math.round((e.seconds ?? 0) / 60)
        toast(min > 0 ? `${e.name}: ${min} min jogados` : `${e.name} fechado`, 'ok')
      }
      app.games = await api.refresh()
    })
    startInput()
    await tick()
    focusDefault()
    api.ready().catch(() => {})
  })

  // preferências de acessibilidade viram classes no <body> e no <html>
  $effect(() => {
    const a = app.settings?.a11y
    if (!a) return
    const body = document.body.classList
    body.toggle('high-contrast', a.highContrast)
    body.toggle('reduce-motion', a.reduceMotion)
    body.toggle('scanlines', a.scanlines && !a.highContrast)
    body.toggle('no-pixel', !a.pixelTitles)
    document.documentElement.style.setProperty('--font-scale', String(a.fontScale))
    setReducedMotion(a.reduceMotion)
    configure(a)
  })

  // ao trocar de tela, foca o padrão dela
  $effect(() => {
    void app.stack.length
    tick().then(focusDefault)
  })
</script>

{#if app.error}
  <main class="fatal">
    <h1 class="pixel">Não deu pra abrir</h1>
    <p>{app.error}</p>
    <button class="btn primary" data-nav data-nav-default onclick={() => location.reload()}>Tentar de novo</button>
  </main>
{:else if !app.ready}
  <main class="loading" aria-busy="true"><p class="pixel">Carregando…</p></main>
{:else}
  {#key app.stack.length + (current.name === 'game' ? current.id : current.name === 'library' ? current.source : '')}
    {#if current.name === 'home'}
      <Home />
    {:else if current.name === 'game'}
      <Game_ id={current.id} />
    {:else if current.name === 'import'}
      <Import />
    {:else if current.name === 'search'}
      <Search />
    {:else if current.name === 'library'}
      <Library source={current.source} />
    {:else if current.name === 'edit'}
      <MetaEditor id={current.id} />
    {:else if current.name === 'settings'}
      <Settings tab={current.tab} />
    {/if}
  {/key}
{/if}

<Playing />
<Tasks />
<Toasts />
<ConfirmDialog />
<PromptDialog />

<style>
  .fatal,
  .loading {
    height: 100vh;
    display: grid;
    place-content: center;
    gap: 1rem;
    text-align: center;
  }
  .loading p {
    font-size: 1.6rem;
    animation: blink 1s steps(2) infinite;
  }
</style>
