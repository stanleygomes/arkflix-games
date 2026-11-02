<script lang="ts">
  // Em quais listas o jogo está: marcar/desmarcar e criar lista nova.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, prompt, putList, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'

  let { gameId, onclose }: { gameId: string; onclose: () => void } = $props()

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.prompt) return false
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })

  async function toggle(id: string, inList: boolean) {
    try {
      putList(await api.setInList(id, gameId, inList))
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function create() {
    await prompt('Nova lista', '', 'Criar lista', async (name) => {
      try {
        const l = await api.createList(name)
        putList(await api.setInList(l.id, gameId, true))
        toast(`Lista "${l.name}" criada`, 'ok', 2500)
        return null
      } catch (err) {
        return (err as Error).message
      }
    })
    tick().then(focusDefault)
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="lists-title" data-nav-scope={app.prompt ? undefined : ''}>
    <h2 id="lists-title">Listas</h2>
    {#if app.lists.length}
      <div class="checks" role="group" aria-label="Listas do jogo">
        {#each app.lists as l, i (l.id)}
          {@const inList = l.games.includes(gameId)}
          <button class="check" role="checkbox" aria-checked={inList} data-nav data-nav-default={i === 0 ? '' : undefined} onclick={() => toggle(l.id, !inList)}>
            <span class="box" aria-hidden="true"></span>{l.name}
            <span class="muted">({l.games.length})</span>
          </button>
        {/each}
      </div>
    {:else}
      <p class="muted">Nenhuma lista ainda.</p>
    {/if}
    <div class="actions">
      <button class="btn" data-nav data-nav-default={app.lists.length ? undefined : ''} onclick={create}><Icon name="plus" /> Nova lista</button>
      <button class="btn primary" data-nav onclick={onclose}>Pronto</button>
    </div>
  </div>
</div>

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    display: grid;
    place-items: center;
    z-index: 80;
  }
  .dialog {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-width: 24rem;
  }
  .checks {
    display: grid;
    gap: 0.4rem;
  }
  .check {
    font: inherit;
    color: var(--text);
    background: color-mix(in srgb, var(--bg) 60%, var(--surface));
    border: 3px solid transparent;
    padding: 0.5rem 0.7rem;
    display: flex;
    gap: 0.6rem;
    align-items: center;
    cursor: pointer;
    text-align: left;
  }
  .box {
    width: 1rem;
    height: 1rem;
    border: 2px solid var(--text);
    flex: none;
  }
  .check[aria-checked='true'] .box {
    background: var(--accent);
  }
  .actions {
    display: flex;
    gap: 0.8rem;
    justify-content: flex-end;
  }
</style>
