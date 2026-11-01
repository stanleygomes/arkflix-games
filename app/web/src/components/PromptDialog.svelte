<script lang="ts">
  // Diálogo de texto: campo + teclado na tela. Voltar cancela.
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import Keyboard from './Keyboard.svelte'

  const p = $derived(app.prompt)
  let previous: HTMLElement | null = null
  let busy = $state(false)
  const typing = $derived(app.family === 'keyboard')

  function close(v: string | null) {
    const cur = app.prompt
    app.prompt = null
    cur?.resolve(v)
    previous?.focus()
  }

  async function submit() {
    if (!app.prompt || busy) return
    busy = true
    const err = await app.prompt.submit(app.prompt.value)
    busy = false
    if (err) {
      if (app.prompt) app.prompt.error = err
      play('error')
      return
    }
    play('accept')
    close(app.prompt?.value ?? null)
  }

  $effect(() => {
    if (!p) return
    previous = document.activeElement as HTMLElement | null
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        close(null)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })
</script>

{#if p}
  <div class="backdrop">
    <div class="dialog panel" role="dialog" aria-modal="true" aria-labelledby="prompt-title" data-nav-scope>
      <h2 id="prompt-title">{p.title}</h2>
      <label class="sr-only" for="prompt-input">{p.title}</label>
      {#if p.multiline}
        <textarea
          id="prompt-input"
          data-nav
          data-nav-default={typing ? '' : undefined}
          bind:value={p.value}
          rows="6"
          maxlength={p.max}
          aria-invalid={p.error ? 'true' : undefined}
          aria-describedby={p.error ? 'prompt-error' : undefined}
        ></textarea>
      {:else}
      <input
        id="prompt-input"
        type={p.secret ? 'password' : 'text'}
        maxlength={p.max}
        data-nav
        data-nav-default={typing ? '' : undefined}
        bind:value={p.value}
        autocomplete="off"
        spellcheck="false"
        aria-invalid={p.error ? 'true' : undefined}
        aria-describedby={p.error ? 'prompt-error' : undefined}
        onkeydown={(e) => {
          if (e.key === 'Enter') {
            e.preventDefault()
            submit()
          }
        }}
      />
      {/if}
      {#if p.error}<p id="prompt-error" class="error" role="alert">{p.error}</p>{/if}
      <Keyboard bind:value={p.value} autofocus={!typing} max={p.max} />
      <div class="actions">
        <button class="btn" data-nav onclick={() => close(null)}>Cancelar</button>
        <button class="btn primary" data-nav disabled={busy} onclick={submit}>{p.ok}</button>
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
    z-index: 110;
  }
  .dialog {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  textarea {
    font: inherit;
    color: var(--text);
    background: color-mix(in srgb, var(--bg) 70%, #000);
    border: 3px solid color-mix(in srgb, var(--text) 30%, var(--surface));
    padding: 0.55rem 0.7rem;
    width: min(40rem, 80vw);
    resize: vertical;
  }
  textarea:focus {
    border-color: var(--accent);
  }
  .error {
    color: #ffb4b4;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.8rem;
  }
</style>
