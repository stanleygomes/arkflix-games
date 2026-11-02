<script lang="ts">
  import { onMount, tick } from 'svelte'
  import Boot from './components/Boot.svelte'
  import ConfirmDialog from './components/ConfirmDialog.svelte'
  import Guide from './components/Guide.svelte'
  import Playing from './components/Playing.svelte'
  import PromptDialog from './components/PromptDialog.svelte'
  import PinDialog from './components/PinDialog.svelte'
  import PixelWipe from './components/PixelWipe.svelte'
  import { screenIn, screenOut, type ScreenMotion } from './lib/motion'
  import PowerMenu from './components/PowerMenu.svelte'
  import Screensaver from './components/Screensaver.svelte'
  import Tasks from './components/Tasks.svelte'
  import Toasts from './components/Toasts.svelte'
  import { api, on } from './lib/api'
  import { startInput } from './lib/input'
  import { watchDownloads } from './lib/downloads.svelte'
  import { watchTasks } from './lib/activity.svelte'
  import { focusDefault, setReducedMotion } from './lib/nav'
  import { configure, pauseMusic, play } from './lib/sound'
  import { app, applyBoot, resetTo, screen, toast } from './lib/state.svelte'
  import type { Game, GameEvent, LimitEvent, Pad } from './lib/types'
  import { watchUpdate } from './lib/update'
  import Restarting from './components/Restarting.svelte'
  import Game_ from './screens/Game.svelte'
  import Home from './screens/Home.svelte'
  import Import from './screens/Import.svelte'
  import Search from './screens/Search.svelte'
  import Library from './screens/Library.svelte'
  import MetaEditor from './screens/MetaEditor.svelte'
  import StatsScreen from './screens/Stats.svelte'
  import Profiles from './screens/Profiles.svelte'
  import ProfileEditor from './screens/ProfileEditor.svelte'
  import Welcome from './screens/Welcome.svelte'
  import You from './screens/You.svelte'
  import Activity from './screens/Activity.svelte'

  const current = $derived(screen())

  // telas "especiais" ligam como uma TV; o resto desliza na direção da navegação
  const CRT = new Set(['you', 'profiles', 'welcome'])
  function motionFor(name: string): ScreenMotion {
    const { dir } = app.nav
    if (dir === 'reset') return { kind: 'none', dir: 1 } // o mosaico em blocos cobre
    if (dir === 'fwd' && CRT.has(name)) return { kind: 'crt', dir: 1 }
    return { kind: 'slide', dir: dir === 'back' ? -1 : 1 }
  }

  onMount(async () => {
    let native = true
    try {
      const boot = await api.bootstrap()
      applyBoot(boot)
      native = boot.nativePads !== false
      if (boot.profile?.setup) resetTo({ name: 'welcome' })
      else if (boot.profile?.choose) resetTo({ name: 'profiles' })
      app.ready = true
      if (boot.pads?.length) app.family = boot.pads[0].family
      if (boot.running) app.playing = { state: 'running', id: boot.running, name: '' }
    } catch (err) {
      app.error = (err as Error).message
    }
    const padsChanged = (pads: Pad[] | null) => {
      const before = app.pads.length
      app.pads = pads ?? []
      if (app.pads.length) app.family = app.pads[0].family
      if (app.pads.length > before) toast(`Controle conectado: ${app.pads[app.pads.length - 1].name}`, 'ok', 3000)
      else if (app.pads.length < before) toast('Controle desconectado', 'info', 3000)
    }
    on<Pad[]>('pads', padsChanged)
    on<Game[]>('library', (games) => (app.games = games ?? []))
    watchUpdate()
    // limite do perfil infantil: aviso faltando pouco e no fim (o jogo fecha salvando)
    on<LimitEvent>('limit', (e) => toast(e.minutesLeft > 0 ? e.message : `${e.message} O jogo foi salvo e fechado.`, 'info', 9000))
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
    startInput(native, padsChanged)
    watchDownloads()
    watchTasks()
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
    document.documentElement.classList.toggle('reduce-motion', a.reduceMotion)
    body.toggle('scanlines', a.scanlines && !a.highContrast)
    body.toggle('no-pixel', !a.pixelTitles)
    // perfil infantil tem o mesmo visual: só os jogos são filtrados
    document.documentElement.style.setProperty('--font-scale', String(a.fontScale))
    setReducedMotion(a.reduceMotion)
    configure(a)
  })

  // som de troca de tela (o da primeira tela é o do boot)
  let lastNav = 0
  $effect(() => {
    const seq = app.nav.seq
    if (seq !== lastNav && lastNav !== 0) play('swoosh')
    lastNav = seq || -1
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
  {#key app.stack.length + current.name + (current.name === 'game' ? current.id : current.name === 'library' ? current.source : '') + (app.profile?.current ?? '')}
    <div class="layer" in:screenIn={motionFor(current.name)} out:screenOut={motionFor(current.name)}>
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
    {:else if current.name === 'stats'}
      <StatsScreen />
    {:else if current.name === 'settings'}
      <You item={current.tab} />
    {:else if current.name === 'profiles'}
      <Profiles />
    {:else if current.name === 'profile'}
      <ProfileEditor id={current.id} setup={current.setup} suggest={current.suggest} />
    {:else if current.name === 'welcome'}
      <Welcome />
    {:else if current.name === 'you'}
      <You item={current.item} />
    {:else if current.name === 'activity'}
      <Activity />
    {/if}
    </div>
  {/key}
{/if}

<Playing />
<Tasks />
<Toasts />
<ConfirmDialog />
<PromptDialog />
<PinDialog />
<PixelWipe />
<PowerMenu />
<Guide />
<Screensaver />
<Restarting />
{#if app.ready}<Boot />{/if}

<style>
  .layer {
    position: fixed;
    inset: 0;
    transform-origin: center;
  }
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
