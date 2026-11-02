<script lang="ts">
  // Configurar este jogo (RetroArch): core, imagem e proporção só pra ele.
  // Cada escolha grava na hora e vale na próxima vez que o jogo abrir.
  import { panelIn, panelOut, shade } from '../lib/motion'
  import { tick } from 'svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { toast } from '../lib/state.svelte'
  import type { GameConfigInfo, GameSettings } from '../lib/types'

  let { gameId, onclose }: { gameId: string; onclose: () => void } = $props()
  let info = $state<GameConfigInfo | null>(null)
  let error = $state('')

  $effect(() => {
    api
      .gameConfig(gameId)
      .then((v) => {
        info = v
        tick().then(focusDefault)
      })
      .catch((err) => (error = (err as Error).message))
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (a === 'back') {
        play('back')
        onclose()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  type Opt = { value: string; label: string }
  const groups = $derived.by(() => {
    if (!info?.supported) return []
    const def = info.cores.find((c) => c.default)
    const cores: Opt[] = [
      { value: '', label: `Padrão${def ? ` (${def.name})` : ''}` },
      ...info.cores.filter((c) => !c.default).map((c) => ({ value: c.id, label: c.name })),
    ]
    const image: Opt[] = [
      { value: '', label: 'Padrão' },
      { value: 'sharp', label: 'Pixels nítidos' },
      { value: 'smooth', label: 'Suave' },
      ...(info.crt ? [{ value: 'crt', label: 'CRT (TV de tubo)' }] : []),
    ]
    const aspect: Opt[] = [
      { value: '', label: 'Padrão do core' },
      { value: '4:3', label: '4:3' },
      { value: 'full', label: 'Tela toda' },
    ]
    return [
      ...(cores.length > 1 ? [{ key: 'core' as const, label: 'Core', opts: cores, hint: 'Pontos salvos de um core não abrem em outro.' }] : []),
      { key: 'image' as const, label: 'Imagem', opts: image, hint: '' },
      { key: 'aspect' as const, label: 'Proporção', opts: aspect, hint: '' },
    ]
  })

  async function pick(key: keyof GameSettings, value: string) {
    if (!info || info.settings[key] === value) return
    try {
      info = await api.setGameConfig(gameId, { ...info.settings, [key]: value } as GameSettings)
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="backdrop" in:shade out:shade={{ out: true }}>
  <div class="dialog panel" in:panelIn out:panelOut role="dialog" aria-modal="true" aria-labelledby="config-title" data-nav-scope>
    <h2 id="config-title">Configurar este jogo</h2>
    {#if error}
      <p class="error" role="alert">{error}</p>
    {:else if !info}
      <p class="muted" role="status">Carregando…</p>
    {:else if !info.supported}
      <p class="muted">Este jogo não abre no RetroArch: configure pelo emulador dele.</p>
    {:else}
      <p class="muted">Só pra este jogo, neste perfil. Vale na próxima vez que ele abrir.</p>
      {#each groups as gr, gi (gr.key)}
        <div class="group">
          <h3 id={`cfg-${gr.key}`}>{gr.label}</h3>
          <div class="opts" role="radiogroup" aria-labelledby={`cfg-${gr.key}`} data-nav-group={`cfg-${gr.key}`}>
            {#each gr.opts as o, oi (o.value)}
              {@const on = (info.settings[gr.key] ?? '') === o.value}
              <button
                class="opt"
                role="radio"
                aria-checked={on}
                data-nav
                data-nav-default={gi === 0 && oi === 0 ? '' : undefined}
                onclick={() => pick(gr.key, o.value)}
              >
                <span class="dot" aria-hidden="true"></span>{o.label}
              </button>
            {/each}
          </div>
          {#if gr.hint}<p class="muted hint">{gr.hint}</p>{/if}
        </div>
      {/each}
    {/if}
    <div class="actions">
      <button class="btn primary" data-nav data-nav-default={info?.supported ? undefined : ''} onclick={onclose}>Pronto</button>
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
    gap: 0.9rem;
    min-width: 26rem;
    max-width: 44rem;
    max-height: 90vh;
    overflow-y: auto;
  }
  h2 {
    font-size: 1.4rem;
  }
  h3 {
    font-size: 1.05rem;
    margin: 0 0 0.4rem;
  }
  .opts {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .opt {
    font: inherit;
    color: var(--text);
    background: color-mix(in srgb, var(--bg) 60%, var(--surface));
    border: 3px solid transparent;
    padding: 0.45rem 0.8rem;
    display: flex;
    gap: 0.5rem;
    align-items: center;
    cursor: pointer;
  }
  .dot {
    width: 0.8rem;
    height: 0.8rem;
    border: 2px solid var(--text);
    border-radius: 50%;
    flex: none;
  }
  .opt[aria-checked='true'] .dot {
    background: var(--accent);
  }
  .hint {
    margin: 0.3rem 0 0;
    font-size: 0.9rem;
  }
  .error {
    color: #ff8a8a;
  }
  .actions {
    display: flex;
    justify-content: flex-end;
  }
</style>
