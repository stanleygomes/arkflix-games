<script lang="ts">
  // Guia rápido (botão Home / G): painel lateral por cima de qualquer tela.
  // Com jogo aberto: voltar ao jogo, salvar/carregar estado, fechar o jogo.
  // Sempre: atalhos (início, central, perfil, busca, você, energia) e som.
  import { tick } from 'svelte'
  import { activity, unread } from '../lib/activity.svelte'
  import { api } from '../lib/api'
  import { closeGuide } from '../lib/guide'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { play } from '../lib/sound'
  import { app, confirm, currentProfile, game, go, isKids, resetTo, screen, toast } from '../lib/state.svelte'
  import type { GuideInfo } from '../lib/types'
  import { saveSoon } from '../screens/settings/save.svelte'
  import Avatar from './Avatar.svelte'
  import Icon, { type IconName } from './Icon.svelte'

  let info = $state<GuideInfo>({ game: '', commands: false })
  let previous: HTMLElement | null = null
  const me = $derived(currentProfile())
  const playing = $derived(app.playing?.state === 'running' ? app.playing : null)
  const current = $derived(playing ? game(playing.id) : undefined)
  const busy = $derived(activity.tasks.filter((t) => t.state === 'running'))
  const news = $derived(unread())
  let now = $state(new Date())

  $effect(() => {
    if (!app.guide) return
    previous = document.activeElement as HTMLElement | null
    now = new Date()
    api.guide().then((g) => (info = g)).catch(() => {})
    tick().then(focusDefault)
    return pushHandler((a) => {
      if (app.confirm) return false
      if (a === 'back' || a === 'guide') {
        play('back')
        close()
        return true
      }
      return a === 'fav' || a === 'details' || a === 'menu' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random'
    })
  })

  function close() {
    closeGuide()
    previous?.focus()
  }

  // atalho: fecha o guia e vai pra tela (sem jogo aberto)
  function open(fn: () => void) {
    app.guide = false
    fn()
  }

  async function command(cmd: 'save' | 'load' | 'quit') {
    if (cmd === 'quit' && !(await confirm(`Fechar ${playing?.name || 'o jogo'}? O que não foi salvo se perde.`, 'Fechar jogo', true))) return
    try {
      await api.gameCommand(cmd)
      app.guide = false
      if (cmd !== 'quit') toast(cmd === 'save' ? 'Estado salvo' : 'Estado carregado', 'ok', 2500)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  function toggle(key: 'music' | 'sounds') {
    if (!app.settings) return
    saveSoon({ ...app.settings, a11y: { ...app.settings.a11y, [key]: !app.settings.a11y[key] } })
  }

  type Tile = { id: string; label: string; icon: IconName; badge?: string; run: () => void }
  const tiles = $derived.by((): Tile[] => {
    if (playing) return []
    const out: Tile[] = [
      { id: 'home', label: 'Início', icon: 'home', run: () => open(() => resetTo({ name: 'home' })) },
      { id: 'activity', label: 'Central', icon: 'bell', badge: busy.length ? `${busy.length} baixando` : news ? `${news} novo${news === 1 ? '' : 's'}` : '', run: () => open(() => screen().name !== 'activity' && go({ name: 'activity' })) },
      { id: 'search', label: 'Buscar', icon: 'search', run: () => open(() => screen().name !== 'search' && go({ name: 'search' })) },
      { id: 'you', label: 'Você', icon: 'gear', run: () => open(() => screen().name !== 'you' && go({ name: 'you' })) },
      { id: 'profiles', label: 'Trocar perfil', icon: 'resume', run: () => open(() => go({ name: 'profiles' })) },
    ]
    if (app.mode === 'desktop') out.push({ id: 'power', label: 'Energia', icon: 'power', run: () => open(() => (app.power = true)) })
    return out
  })
</script>

{#if app.guide}
  <div class="backdrop" role="presentation" onclick={(e) => e.target === e.currentTarget && close()}>
    <div class="guide" role="dialog" aria-modal="true" aria-labelledby="guide-title" data-nav-scope={app.confirm ? undefined : ''}>
      <header>
        {#if me}<Avatar avatar={me.avatar} color={me.color} size="3rem" />{/if}
        <div>
          <h2 id="guide-title" class="pixel">{me?.name ?? 'Guia'}</h2>
          <p class="muted">{now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' })} · {app.pads.length ? app.pads.map((p) => p.name).join(', ') : 'sem controle'}</p>
        </div>
      </header>

      {#if playing}
        <section aria-label="Jogo aberto" class="game">
          {#if current?.cover}<img src={current.cover} alt="" />{/if}
          <p class="now pixel">Jogando</p>
          <p class="name">{playing.name || current?.name}</p>
          <div class="actions">
            <button class="btn primary" data-nav data-nav-default onclick={close}><Icon name="play" /> Voltar ao jogo</button>
            {#if info.commands}
              <button class="btn" data-nav onclick={() => command('save')}><Icon name="save" /> Salvar estado</button>
              <button class="btn" data-nav onclick={() => command('load')}><Icon name="resume" /> Carregar estado</button>
            {/if}
            <button class="btn danger" data-nav onclick={() => command('quit')}><Icon name="close" /> Fechar jogo</button>
          </div>
        </section>
      {:else}
        <nav aria-label="Atalhos" class="tiles">
          {#each tiles as t, i (t.id)}
            <button class="tile" data-nav data-nav-default={i === 0 ? '' : undefined} onclick={t.run}>
              <Icon name={t.icon} size="1.5em" />
              <span>{t.label}</span>
              {#if t.badge}<span class="badge pixel">{t.badge}</span>{/if}
            </button>
          {/each}
        </nav>
      {/if}

      {#if busy.length}
        <section aria-label="Em andamento" class="busy">
          {#each busy.slice(0, 2) as t (t.id)}
            <p class="muted">{t.label} · {Math.round(t.progress * 100)}%</p>
            <div class="energy" role="progressbar" aria-label={t.label} aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(t.progress * 100)}>
              <span style:width={`${Math.floor(t.progress * 20) * 5}%`}></span>
            </div>
          {/each}
        </section>
      {/if}

      {#if app.settings && !isKids()}
        <section aria-label="Som" class="quick">
          <button class="btn small" data-nav role="switch" aria-checked={app.settings.a11y.music} onclick={() => toggle('music')}>Música</button>
          <button class="btn small" data-nav role="switch" aria-checked={app.settings.a11y.sounds} onclick={() => toggle('sounds')}>Sons do menu</button>
        </section>
      {/if}
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    z-index: 95;
    background: rgba(0, 0, 0, 0.55);
  }
  .guide {
    position: absolute;
    inset: 0 auto 0 0;
    width: min(26rem, 92vw);
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
    padding: 1.6rem 1.5rem;
    background: var(--surface);
    border-right: 3px solid var(--accent);
    box-shadow: 0 0 24px color-mix(in srgb, var(--accent) 60%, transparent);
    animation: guide-in 200ms var(--ease);
  }
  @keyframes guide-in {
    from {
      transform: translateX(-100%);
    }
  }
  :global(.reduce-motion) .guide {
    animation: none;
  }
  header {
    display: flex;
    gap: 0.9rem;
    align-items: center;
  }
  h2 {
    font-size: 1.2rem;
  }
  header p {
    margin: 0.2rem 0 0;
    font-size: 0.85rem;
  }
  .game {
    display: grid;
    gap: 0.6rem;
  }
  .game img {
    width: 100%;
    max-height: 9rem;
    object-fit: cover;
    image-rendering: pixelated;
    border: 2px solid var(--accent-2);
  }
  .now {
    color: var(--focus);
    font-size: 0.8rem;
    margin: 0;
  }
  .name {
    font-weight: 700;
    font-size: 1.2rem;
    margin: 0;
  }
  .actions,
  .tiles {
    display: grid;
    gap: 0.6rem;
  }
  .actions .btn {
    justify-content: flex-start;
  }
  .tiles {
    grid-template-columns: 1fr 1fr;
  }
  .tile {
    display: grid;
    justify-items: start;
    gap: 0.4rem;
    padding: 0.9rem;
    font: inherit;
    font-weight: 700;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--surface) 70%, #000);
    border: 2px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
    cursor: pointer;
  }
  .tile :global(svg) {
    color: var(--accent);
  }
  .badge {
    font-size: 0.65rem;
    color: var(--focus);
  }
  .busy p {
    margin: 0.4rem 0 0.2rem;
    font-size: 0.85rem;
  }
  .quick {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
    margin-top: auto;
  }
</style>
