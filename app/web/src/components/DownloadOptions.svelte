<script lang="ts">
  // Opções de download (Central): velocidade máxima e "só de madrugada".
  // Cada escolha já vale; prende o foco ([data-nav-scope]); voltar fecha.
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, toast } from '../lib/state.svelte'
  import type { DownloadOptions } from '../lib/types'

  let { onclose }: { onclose: () => void } = $props()

  const SPEEDS = [
    { kbps: 0, label: 'Sem limite' },
    { kbps: 1024, label: '1 MB/s' },
    { kbps: 5120, label: '5 MB/s' },
    { kbps: 10240, label: '10 MB/s' },
  ]
  const opts = $derived<DownloadOptions>(app.settings?.downloads ?? { limitKBps: 0, nightOnly: false })

  async function save(next: DownloadOptions) {
    try {
      app.settings = await api.setDownloadOptions(next)
      play('accept')
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back' || a === 'menu') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'search'
    })
  })
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="menu panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="dl-options-title" data-nav-scope>
    <h2 id="dl-options-title">Opções de download</h2>
    <fieldset>
      <legend>Velocidade</legend>
      <div class="options" role="radiogroup" aria-label="Velocidade">
        {#each SPEEDS as s (s.kbps)}
          <button
            class="option"
            role="radio"
            aria-checked={opts.limitKBps === s.kbps}
            data-nav
            data-nav-default={opts.limitKBps === s.kbps ? '' : undefined}
            onclick={() => save({ ...opts, limitKBps: s.kbps })}
          >
            <span class="dot" aria-hidden="true"></span>{s.label}
          </button>
        {/each}
      </div>
    </fieldset>
    <button class="option" role="switch" aria-checked={opts.nightOnly} aria-describedby="dl-night-hint" data-nav onclick={() => save({ ...opts, nightOnly: !opts.nightOnly })}>
      <span class="box" aria-hidden="true"></span>Só de madrugada
    </button>
    <p id="dl-night-hint" class="hint muted">Os downloads esperam na fila e começam entre 0h e 6h.</p>
    <div class="actions">
      <button class="btn primary" data-nav onclick={onclose}>Fechar</button>
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
  .menu {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-width: 22rem;
    max-height: 90vh;
    overflow-y: auto;
  }
  fieldset {
    border: none;
    margin: 0;
    padding: 0;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.6rem;
  }
  .options {
    display: grid;
    gap: 0.3rem;
  }
  .option {
    font: inherit;
    color: var(--text);
    background: transparent;
    border: 3px solid transparent;
    text-align: left;
    padding: 0.45rem 0.6rem;
    display: flex;
    gap: 0.7rem;
    align-items: center;
    cursor: pointer;
  }
  .dot,
  .box {
    width: 0.9rem;
    height: 0.9rem;
    border: 2px solid var(--text);
    flex: none;
  }
  .option[aria-checked='true'] .dot,
  .option[aria-checked='true'] .box {
    background: var(--accent);
    box-shadow: inset 0 0 0 2px var(--surface);
  }
  .hint {
    margin: -0.6rem 0 0;
    font-size: 0.85rem;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
  }
</style>
