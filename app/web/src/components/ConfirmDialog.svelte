<script lang="ts">
  // Diálogo de confirmação. [data-nav-scope] prende a navegação aqui dentro;
  // "voltar" cancela.
  import { tick } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { app } from '../lib/state.svelte'

  const c = $derived(app.confirm)
  let previous: HTMLElement | null = null

  function close(v: boolean) {
    const cur = app.confirm
    app.confirm = null
    cur?.resolve(v)
    previous?.focus()
  }

  $effect(() => {
    if (!c) return
    previous = document.activeElement as HTMLElement | null
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        close(false)
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })
</script>

{#if c}
  <div class="backdrop">
    <div class="dialog panel" role="alertdialog" aria-modal="true" aria-labelledby="confirm-text" data-nav-scope>
      <p id="confirm-text" class="text">{c.text}</p>
      <div class="actions">
        <button class="btn" data-nav data-nav-default onclick={() => close(false)}>Cancelar</button>
        <button class="btn" class:danger={c.danger} class:primary={!c.danger} data-nav onclick={() => close(true)}>{c.ok}</button>
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
    z-index: 100;
  }
  .dialog {
    max-width: 32rem;
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
  }
  .text {
    font-size: 1.15rem;
  }
  .actions {
    display: flex;
    gap: 1rem;
    justify-content: flex-end;
  }
</style>
