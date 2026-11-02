<script lang="ts">
  // Opções do jogo: listas, editar, configurar, ocultar, apagar. Tudo que mexe
  // no console fica aqui dentro, longe dos botões principais. Com quick (Start
  // num jogo do Início), vem antes o básico: jogar/baixar, continuar,
  // favoritar e detalhes.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { bytes } from '../lib/format'
  import { app, back, confirm, game, go, isKids, putGame, system, toast } from '../lib/state.svelte'
  import Icon from './Icon.svelte'
  import { toggleFavorite } from '../lib/actions'
  import { downloadGame } from '../lib/downloads.svelte'

  let {
    gameId,
    onclose,
    onlists,
    onconfig,
    quick = false,
  }: { gameId: string; onclose: () => void; onlists: () => void; onconfig: () => void; quick?: boolean } = $props()

  async function launch(state = '') {
    if (!g) return
    onclose()
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  function details() {
    onclose()
    go({ name: 'game', id: gameId })
  }
  const g = $derived(game(gameId))
  const canResume = $derived(!!g && !g.remote && g.lastPlayed > 0 && system(g.system)?.emulator === 'retroarch')
  const configurable = $derived(!!g && !g.remote && system(g.system)?.emulator === 'retroarch')
  const inLists = $derived(app.lists.filter((l) => l.games.includes(gameId)).length)

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm || app.prompt) return false
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  function edit() {
    onclose()
    go({ name: 'edit', id: gameId })
  }

  async function toggleHidden() {
    if (!g) return
    try {
      const updated = await api.setHidden(gameId, !g.hidden)
      putGame(updated)
      play('toggle')
      toast(updated.hidden ? 'Jogo oculto. Pra mostrar de novo: configurações › Biblioteca › Jogos ocultos.' : 'Jogo de volta na biblioteca', 'ok', 4000)
      onclose()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function remove() {
    if (!g) return
    try {
      const info = await api.gameSize(gameId)
      const after = info.server ? ' Ele continua no servidor e dá pra baixar de novo.' : ''
      const ok = await confirm(
        `Apagar "${g.name}" do PC? Libera ${bytes(info.size)}. Os saves e pontos salvos não são apagados.${after}`,
        'Apagar do PC',
        true,
      )
      if (!ok) return
      await api.deleteGame(gameId)
      toast(`"${g.name}" apagado do PC`, 'ok', 3000)
      play('accept')
      onclose()
      if (!info.server) {
        app.games = app.games.filter((x) => x.id !== gameId)
        back()
      }
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div
    class="dialog panel"
    in:panelIn
    out:panelOut
    role="dialog"
    aria-modal="true"
    aria-labelledby="options-title"
    data-nav-scope={app.confirm || app.prompt ? undefined : ''}
  >
    <h2 id="options-title">Opções</h2>
    <p class="muted name">{g?.name}</p>
    <div class="options">
      {#if quick && g}
        {#if g.remote}
          {#if !isKids()}<button class="btn primary" data-nav data-nav-default onclick={() => (onclose(), downloadGame(g))}><Icon name="download" /> Baixar do servidor</button>{/if}
        {:else}
          <button class="btn primary" data-nav data-nav-default onclick={() => launch()}><Icon name="play" /> Jogar</button>
          {#if canResume}<button class="btn" data-nav onclick={() => launch('auto')}><Icon name="resume" /> Continuar de onde parei</button>{/if}
        {/if}
        <button class="btn" data-nav onclick={() => (toggleFavorite(gameId), onclose())}><Icon name="star" /> {g.favorite ? 'Tirar dos favoritos' : 'Favoritar'}</button>
        <button class="btn" data-nav onclick={details}><Icon name="disc" /> Informações do jogo</button>
        <hr />
      {/if}
      <button class="btn" data-nav data-nav-default={quick ? undefined : ''} aria-haspopup="dialog" onclick={onlists}>
        <Icon name="plus" /> Listas{inLists ? ` (${inLists})` : ''}
      </button>
      {#if !isKids()}
        <button class="btn" data-nav onclick={edit}><Icon name="pencil" /> Editar informações</button>
        {#if configurable}
          <button class="btn" data-nav aria-haspopup="dialog" onclick={onconfig}><Icon name="gear" /> Configurar este jogo</button>
        {/if}
        <button class="btn" data-nav onclick={toggleHidden}>
          <Icon name={g?.hidden ? 'check' : 'close'} /> {g?.hidden ? 'Mostrar na biblioteca' : 'Ocultar da biblioteca'}
        </button>
        {#if g?.local && !g.remote}
          <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar do PC</button>
        {/if}
      {/if}
      <button class="btn" data-nav onclick={onclose}>Cancelar</button>
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
    display: grid;
    gap: 0.8rem;
    min-width: 22rem;
    padding: 1.4rem 1.8rem;
  }
  h2 {
    font-size: 1.4rem;
  }
  .name {
    margin: 0;
  }
  .options {
    display: grid;
    gap: 0.7rem;
  }
  .options .btn {
    justify-content: flex-start;
  }
  hr {
    width: 100%;
    border: none;
    border-top: 2px solid color-mix(in srgb, var(--accent-2) 30%, transparent);
    margin: 0.2rem 0;
  }
</style>
