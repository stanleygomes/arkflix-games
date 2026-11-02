<script lang="ts">
  // Teclado de PIN (4 números): controle navega no teclado da tela, teclado
  // físico digita direto. Com 4 números confere; errou, limpa e avisa.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  const ask = $derived(app.pin)
  let digits = $state('')
  let busy = $state(false)
  let previous: HTMLElement | null = null

  function close(v: string | null) {
    const cur = app.pin
    app.pin = null
    digits = ''
    cur?.resolve(v)
    previous?.focus()
  }

  async function type(d: string) {
    if (busy || digits.length >= 4) return
    digits += d
    play('move')
    if (digits.length < 4 || !app.pin) return
    busy = true
    const err = await app.pin.submit(digits)
    busy = false
    if (!app.pin) return
    if (err) {
      play('error')
      app.pin.error = err
      digits = ''
      return
    }
    play('accept')
    close(digits)
  }

  function erase() {
    digits = digits.slice(0, -1)
  }

  $effect(() => {
    if (!ask) return
    previous = document.activeElement as HTMLElement | null
    digits = ''
    tick().then(focusDefault)
    const onKey = (ev: KeyboardEvent) => {
      if (/^[0-9]$/.test(ev.key)) {
        ev.preventDefault()
        ev.stopPropagation()
        type(ev.key)
      } else if (ev.key === 'Backspace') {
        ev.preventDefault()
        erase()
      }
    }
    window.addEventListener('keydown', onKey, { capture: true })
    const off = pushHandler((a) => {
      if (a === 'back') {
        play('back')
        close(null)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
    return () => {
      window.removeEventListener('keydown', onKey, { capture: true })
      off()
    }
  })
</script>

{#if ask}
  <div class="backdrop" in:shade out:shade={{ out: true }}>
    <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="pin-title" data-nav-scope>
      <h2 id="pin-title">{ask.title}</h2>
      <p class="dots" aria-live="polite">
        <span class="sr-only">{digits.length} de 4 números</span>
        {#each [0, 1, 2, 3] as i (i)}<span class="dot" class:on={i < digits.length} aria-hidden="true"></span>{/each}
      </p>
      {#if ask.error}<p class="error" role="alert">{ask.error}</p>{/if}
      <div class="keys">
        {#each ['1', '2', '3', '4', '5', '6', '7', '8', '9'] as d (d)}
          <button class="btn key pixel" data-nav data-nav-default={d === '1' ? '' : undefined} onclick={() => type(d)}>{d}</button>
        {/each}
        <button class="btn key" data-nav onclick={erase} aria-label="Apagar número"><Icon name="back" /></button>
        <button class="btn key pixel" data-nav onclick={() => type('0')}>0</button>
        <button class="btn key" data-nav onclick={() => close(null)}>Cancelar</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.7);
    display: grid;
    place-items: center;
    z-index: 95;
  }
  .dialog {
    display: grid;
    gap: 1rem;
    justify-items: center;
    padding: 1.6rem 2rem;
  }
  h2 {
    font-size: 1.4rem;
  }
  .dots {
    display: flex;
    gap: 0.8rem;
    margin: 0;
  }
  .dot {
    width: 1.2rem;
    height: 1.2rem;
    border: 3px solid var(--text);
  }
  .dot.on {
    background: var(--accent);
    border-color: var(--accent);
  }
  .error {
    color: #ff9c9c;
    font-weight: 700;
  }
  .keys {
    display: grid;
    grid-template-columns: repeat(3, 5.5rem);
    gap: 0.6rem;
  }
  .key {
    justify-content: center;
    font-size: 1.3rem;
    min-height: 3.2rem;
  }
</style>
