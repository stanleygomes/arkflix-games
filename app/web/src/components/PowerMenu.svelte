<script lang="ts">
  // Menu de energia (só no app desktop): minimizar, tela cheia/janela, fechar.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, confirm, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let fullscreen = $state(true)
  let previous: HTMLElement | null = null

  function close() {
    app.power = false
    previous?.focus()
  }

  $effect(() => {
    if (!app.power) return
    previous = document.activeElement as HTMLElement | null
    api.isFullscreen().then((v) => (fullscreen = v)).catch(() => {})
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm) return false
      if (a === 'back') {
        play('back')
        close()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  async function run(fn: () => Promise<unknown>) {
    try {
      await fn()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function minimize() {
    close()
    await run(api.minimize)
  }

  async function toggle() {
    close()
    await run(api.toggleFullscreen)
  }

  async function quit() {
    if (!(await confirm('Fechar o arkflix-games?', 'Fechar'))) return
    close()
    await run(api.quit)
  }
</script>

{#if app.power}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="power-title" data-nav-scope={app.confirm ? undefined : ''}>
      <h2 id="power-title">Energia</h2>
      <div class="options">
        <button class="btn" data-nav data-nav-default onclick={minimize}><Icon name="back" /> Minimizar</button>
        <button class="btn" data-nav onclick={toggle}><Icon name="sliders" /> {fullscreen ? 'Sair da tela cheia' : 'Tela cheia'}</button>
        <button class="btn danger" data-nav onclick={quit}><Icon name="power" /> Fechar o app</button>
        <button class="btn" data-nav onclick={close}>Cancelar</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    display: grid;
    place-items: center;
    z-index: 85;
  }
  .dialog {
    display: grid;
    gap: 1rem;
    min-width: 20rem;
    padding: 1.4rem 1.8rem;
  }
  h2 {
    font-size: 1.4rem;
  }
  .options {
    display: grid;
    gap: 0.7rem;
  }
  .options .btn {
    justify-content: flex-start;
  }
</style>
