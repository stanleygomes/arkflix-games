<script lang="ts">
  // Mover os jogos de um console pra outra pasta da biblioteca (outro disco).
  // Escolhe o destino aqui; o progresso fica na Central.
  import { tick, untrack } from 'svelte'
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { bytes } from '../../lib/format'
  import { pushHandler } from '../../lib/input'
  import { panelIn, panelOut, shade } from '../../lib/motion'
  import { focusDefault } from '../../lib/nav'
  import { play } from '../../lib/sound'
  import { app, toast } from '../../lib/state.svelte'
  import type { MoveTarget } from '../../lib/types'

  let { system, name, roots, onclose }: { system: string; name: string; roots: string[]; onclose: () => void } = $props()

  const label = (path: string) => app.settings?.libraryDirs.find((d) => d.path === path)?.label || path
  let from = $state(untrack(() => roots[0] ?? ''))
  let targets = $state<MoveTarget[] | null>(null)
  let to = $state('')
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    const src = from
    targets = null
    error = ''
    api
      .movePlan(system, src)
      .then((t) => {
        targets = t
        to = t.find((x) => x.fits)?.path ?? ''
      })
      .catch((err) => (error = (err as Error).message))
      .finally(() => tick().then(focusDefault))
  })

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab'
    })
  })

  async function start() {
    busy = true
    try {
      await api.moveGames(system, from, to)
      toast(`Movendo os jogos de ${name}. Acompanhe na Central.`, 'info', 4000)
      onclose()
    } catch (err) {
      error = (err as Error).message
    } finally {
      busy = false
    }
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="move-title" data-nav-scope>
    <h2 id="move-title">Mover jogos de {name}</h2>
    {#if roots.length > 1}
      <div class="choices" role="radiogroup" aria-label="De qual pasta">
        <span class="muted small">De</span>
        {#each roots as r (r)}
          <button class="check" role="radio" aria-checked={from === r} data-nav onclick={() => (from = r)}>
            <span class="box" aria-hidden="true"></span>{label(r)}
          </button>
        {/each}
      </div>
    {:else}
      <p class="muted small">De {label(from)} <span class="path">({from})</span></p>
    {/if}

    {#if !targets && !error}
      <p class="muted" aria-busy="true">Medindo…</p>
    {:else if targets && targets.length === 0}
      <p class="muted">Nenhuma outra pasta gravável na biblioteca. Marque uma como gravável em Biblioteca.</p>
    {:else if targets}
      <p class="muted small">{targets[0].count} jogo{targets[0].count === 1 ? '' : 's'}, {bytes(targets[0].size)}</p>
      <div class="choices" role="radiogroup" aria-label="Para qual pasta">
        {#each targets as t, i (t.path)}
          <button
            class="check"
            role="radio"
            aria-checked={to === t.path}
            disabled={!t.fits}
            data-nav
            data-nav-default={t.path === to || (!to && i === 0) ? '' : undefined}
            onclick={() => (to = t.path)}
          >
            <span class="box" aria-hidden="true"></span>
            <span class="target">
              {t.label}
              <span class="muted small path">{t.path}</span>
            </span>
            <span class="muted small">{t.sameDisk ? 'mesmo disco' : t.free < 0 ? '' : `${bytes(t.free)} livres`}{t.fits ? '' : ' · sem espaço'}</span>
          </button>
        {/each}
      </div>
    {/if}
    {#if error}<p class="error" role="alert">{error}</p>{/if}

    <div class="actions">
      <button class="btn" data-nav onclick={onclose}>Cancelar</button>
      <button class="btn primary" data-nav disabled={!to || busy} onclick={start}><Icon name="folder" /> Mover</button>
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
    max-width: min(40rem, 92vw);
  }
  h2 {
    font-size: 1.3rem;
  }
  .choices {
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
  .check:disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }
  .target {
    display: flex;
    flex-direction: column;
    flex: 1;
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
  .small {
    font-size: 0.85rem;
  }
  .path {
    overflow-wrap: anywhere;
  }
  .error {
    color: #ff9c9c;
  }
  .actions {
    display: flex;
    gap: 0.8rem;
    justify-content: flex-end;
  }
</style>
