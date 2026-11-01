<script lang="ts">
  import { tick } from 'svelte'
  import GameArt from '../components/GameArt.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { toggleFavorite } from '../lib/actions'
  import { sortGames } from '../lib/library'
  import { activeCollections, sortOf, sourceGames, sourceTitle } from '../lib/sources'
  import { app, go, system, toast } from '../lib/state.svelte'
  import type { Game } from '../lib/types'

  type Tab = { slug: string; short: string; name: string; kind: 'home' | 'collection' | 'system' }
  const tabs = $derived<Tab[]>([
    { slug: 'home', short: 'Início', name: 'Início', kind: 'home' },
    ...activeCollections(app.games).map((c) => ({ slug: `collection:${c.id}`, short: c.short, name: c.label, kind: 'collection' as const })),
    ...app.systems
      .filter((s) => app.games.some((g) => g.system === s.slug))
      .map((s) => ({ slug: s.slug, short: s.short, name: s.name, kind: 'system' as const })),
  ])

  type Row = { id: string; title: string; games: Game[]; source?: string }
  const rows = $derived.by((): Row[] => {
    if (app.homeTab.startsWith('collection:')) {
      const games = sortGames(sourceGames(app.homeTab), sortOf(app.homeTab))
      return [{ id: app.homeTab, title: `${sourceTitle(app.homeTab)} · ${games.length} jogo${games.length === 1 ? '' : 's'}`, games, source: app.homeTab }]
    }
    if (app.homeTab !== 'home') {
      const sys = system(app.homeTab)
      const games = sortGames(
        app.games.filter((g) => g.system === app.homeTab),
        sortOf(`system:${app.homeTab}`),
      )
      return [{ id: app.homeTab, title: `${sys?.name ?? ''} · ${games.length} jogo${games.length === 1 ? '' : 's'}`, games, source: `system:${app.homeTab}` }]
    }
    const out: Row[] = []
    const recent = app.games.filter((g) => g.lastPlayed > 0).sort((a, b) => b.lastPlayed - a.lastPlayed).slice(0, 12)
    if (recent.length) out.push({ id: 'recent', title: 'Continuar jogando', games: recent })
    const favs = app.games.filter((g) => g.favorite)
    if (favs.length) out.push({ id: 'favorites', title: 'Favoritos', games: favs })
    if (!out.length) out.push({ id: 'all', title: `Sua biblioteca · ${app.games.length} jogos`, games: app.games })
    return out
  })

  const selected = $derived(app.games.find((g) => g.id === app.selected) ?? rows[0]?.games[0])
  const selSystem = $derived(selected ? system(selected.system) : undefined)
  const canResume = $derived(!!selected && selected.lastPlayed > 0 && selSystem?.emulator === 'retroarch')

  // paleta: a do console da aba; no Início, a do jogo selecionado
  $effect(() => {
    if (app.settings?.a11y.highContrast) return applyPalette(highContrastPalette)
    const sys = app.homeTab === 'home' || app.homeTab.startsWith('collection:') ? selSystem : system(app.homeTab)
    applyPalette(sys?.palette ?? homePalette)
  })

  function selectTab(slug: string) {
    if (app.homeTab === slug) return
    app.homeTab = slug
    app.selected = ''
  }

  async function launch(g: Game | undefined, state = '') {
    if (!g) return
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  const toggleFav = (g: Game | undefined) => toggleFavorite(g?.id)

  function focusedGame(): Game | undefined {
    const id = current()?.dataset.game
    return id ? app.games.find((g) => g.id === id) : selected
  }

  function cycleTab(delta: number) {
    const i = tabs.findIndex((t) => t.slug === app.homeTab)
    const next = tabs[(i + delta + tabs.length) % tabs.length]
    selectTab(next.slug)
    tick().then(() => focusEl(document.querySelector<HTMLElement>(`[data-tab="${next.slug}"]`)))
    play('move')
  }

  $effect(() =>
    pushHandler((a) => {
      const inTabs = !!current()?.closest('.tabs')
      switch (a) {
        case 'down':
          if (inTabs) {
            // das abas direto pra fileira de jogos (o destaque fica no meio)
            const card = document.querySelector<HTMLElement>('.shelf [data-game]')
            if (card) {
              focusEl(document.querySelector<HTMLElement>(`.shelf [data-game="${app.selected}"]`) ?? card)
              play('move')
              return true
            }
          }
          return false
        case 'up':
          if (current()?.closest('.shelf')) {
            const playBtn = document.querySelector<HTMLElement>('.hero .actions [data-nav]')
            if (playBtn) {
              focusEl(playBtn)
              play('move')
              return true
            }
          }
          return false
        case 'fav':
          toggleFav(focusedGame())
          return true
        case 'details': {
          const g = focusedGame()
          if (g) go({ name: 'game', id: g.id })
          return true
        }
        case 'menu':
          go({ name: 'settings' })
          return true
        case 'prevTab':
          cycleTab(-1)
          return true
        case 'nextTab':
          cycleTab(1)
          return true
      }
      return false
    }),
  )

  const hero = $derived(selected?.cover ?? '')
</script>

<div class="home">
  <div class="backdrop" style:background-image={hero ? `url("${hero}")` : 'none'} aria-hidden="true"></div>
  <TopBar>
    <nav class="tabs" aria-label="Consoles">
      {#each tabs as t (t.slug)}
        <button
          class="tab pixel"
          class:collection={t.kind === 'collection'}
          class:first-system={t.kind === 'system' && tabs.find((x) => x.kind === 'system') === t}
          data-nav
          data-tab={t.slug}
          data-nav-group="tabs"
          data-nav-default={t.slug === app.homeTab ? '' : undefined}
          aria-current={t.slug === app.homeTab ? 'page' : undefined}
          onfocus={() => selectTab(t.slug)}
          onclick={() => selectTab(t.slug)}
        >
          {t.short}{#if t.name !== t.short}<span class="sr-only"> ({t.name})</span>{/if}
        </button>
      {/each}
    </nav>
  </TopBar>

  <main class="content" aria-labelledby="hero-title">
    {#if app.games.length === 0}
      <section class="empty panel">
        <h1 class="pixel" id="hero-title">Nenhum jogo ainda</h1>
        <p class="muted">Importe ROMs ou aponte suas pastas de jogos nas configurações.</p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'import' })}><Icon name="plus" /> Importar jogos</button>
          <button class="btn" data-nav onclick={() => go({ name: 'settings' })}><Icon name="gear" /> Configurações</button>
        </div>
      </section>
    {:else if selected}
      <div class="stage">
      <section class="hero">
        <p class="system-pill pixel">{selSystem?.name ?? selected.system}</p>
        <h1 class="pixel title" id="hero-title">{selected.name}</h1>
        <p class="meta">
          {#if selected.year}<span>{selected.year}</span>{/if}
          {#if selected.players}<span>{selected.players} jogador{selected.players === '1' ? '' : 'es'}</span>{/if}
          {#each selected.genres.slice(0, 2) as genre (genre)}<span>{genre}</span>{/each}
          {#if selected.favorite}<span class="fav"><Icon name="star" /> Favorito</span>{/if}
        </p>
        {#if selected.summary}<p class="summary">{selected.summary}</p>{/if}
        <div class="actions">
          <button class="btn primary" data-nav data-nav-group="actions" onclick={() => launch(selected)}>
            <Icon name="play" /> Jogar
          </button>
          {#if canResume}
            <button class="btn" data-nav data-nav-group="actions" onclick={() => launch(selected, 'auto')}>
              <Icon name="resume" /> Continuar
            </button>
          {/if}
          <button class="btn" data-nav data-nav-group="actions" aria-pressed={selected.favorite} onclick={() => toggleFav(selected)}>
            <Icon name="star" /> {selected.favorite ? 'Favorito' : 'Favoritar'}
          </button>
          <button class="btn" data-nav data-nav-group="actions" onclick={() => go({ name: 'game', id: selected.id })}>
            <Icon name="disc" /> Saves e detalhes
          </button>
        </div>
      </section>
      <div class="hero-art" aria-hidden="true">
        {#key selected.id}<GameArt game={selected} system={selSystem} large />{/key}
      </div>
      </div>

      <div class="shelves">
        {#each rows as row (row.id)}
          <section class="shelf" aria-label={row.title}>
            <h2 class="row-title">{row.title}</h2>
            <ul class="row" data-nav-group={`row-${row.id}`}>
              {#each row.games as g (g.id)}
                {@const sys = system(g.system)}
                <li>
                  <button
                    class="card"
                    class:selected={g.id === selected.id}
                    data-nav
                    data-nav-inline="center"
                    data-game={g.id}
                    aria-label={`${g.name}, ${sys?.short ?? g.system}${g.year ? `, ${g.year}` : ''}${g.favorite ? ', favorito' : ''}`}
                    onfocus={() => (app.selected = g.id)}
                    onclick={() => go({ name: 'game', id: g.id })}
                  >
                    <GameArt game={g} system={sys} />
                    {#if g.favorite}<span class="badge" aria-hidden="true"><Icon name="star" /></span>{/if}
                  </button>
                </li>
              {/each}
              {#if row.source}
                <li>
                  <button class="card see-all pixel" data-nav data-nav-inline="center" onclick={() => go({ name: 'library', source: row.source! })}>
                    <Icon name="sliders" size="1.6em" /> Ver tudo
                  </button>
                </li>
              {/if}
            </ul>
          </section>
        {/each}
      </div>
    {/if}
  </main>

  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Favorito' },
      { button: 'west', label: 'Detalhes' },
      { button: 'l1', label: 'Console anterior' },
      { button: 'r1', label: 'Próximo console' },
      { button: 'select', label: 'Buscar' },
      { button: 'start', label: 'Configurações' },
    ]}
  />
</div>

<style>
  .home {
    height: 100vh;
    display: flex;
    flex-direction: column;
    position: relative;
  }
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center 30%;
    image-rendering: pixelated;
    filter: blur(2px) saturate(1.1);
    opacity: 0.32;
    transform: scale(1.05);
    z-index: -1;
    mask-image: linear-gradient(to bottom, #000 30%, transparent 85%);
  }
  .tabs {
    display: flex;
    gap: 0.4rem;
    overflow-x: auto;
    scrollbar-width: none;
    padding: 0.5rem;
  }
  .tab {
    font-size: 1.05rem;
    color: var(--muted);
    background: transparent;
    border: 3px solid transparent;
    padding: 0.4rem 0.9rem;
    cursor: pointer;
    white-space: nowrap;
  }
  .tab.collection {
    font-family: var(--font-body);
    font-weight: 700;
    font-size: 0.95rem;
  }
  .tab.first-system {
    margin-left: 0.8rem;
    box-shadow: -0.8rem 0 0 -0.7rem color-mix(in srgb, var(--text) 40%, transparent);
  }
  .tab[aria-current='page'] {
    color: var(--on-accent);
    background: var(--accent);
    border-color: #05040a;
    box-shadow: 3px 3px 0 var(--shadow);
  }
  .content {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    min-height: 0;
    padding: 0 2.5rem;
    overflow-y: auto;
  }
  .stage {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: 2rem;
    flex: 1;
    min-height: 0;
  }
  .hero-art {
    width: clamp(10rem, 18vw, 17rem);
    flex: none;
    margin-top: 2vh;
    transform: rotate(-2deg);
    filter: drop-shadow(10px 10px 0 var(--shadow));
    animation: rise 260ms var(--ease);
  }
  @keyframes rise {
    from {
      transform: rotate(-2deg) translateY(16px);
      opacity: 0;
    }
  }
  @media (max-height: 720px) {
    .hero-art {
      display: none;
    }
  }
  .hero {
    max-width: 52rem;
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
    padding-top: 2vh;
  }
  .system-pill {
    color: var(--accent-2);
    font-size: 1rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  .title {
    font-size: clamp(2rem, 4.5vw, 3.6rem);
    line-height: 1.05;
    text-shadow: 4px 4px 0 var(--shadow);
  }
  .meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem 1rem;
    color: var(--muted);
  }
  .meta .fav {
    color: var(--accent);
    display: inline-flex;
    gap: 0.3rem;
    align-items: center;
  }
  .summary {
    max-width: 44rem;
    color: var(--text);
    display: -webkit-box;
    -webkit-line-clamp: 3;
    line-clamp: 3;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
    margin-top: 0.5rem;
  }
  .shelves {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding-bottom: 0.5rem;
  }
  .row-title {
    font-size: 1rem;
    color: var(--muted);
    margin-bottom: 0.5rem;
    font-family: var(--font-body);
  }
  .row {
    list-style: none;
    margin: 0;
    padding: 0.8rem 0.5rem 1.2rem;
    display: flex;
    gap: 1.1rem;
    overflow-x: auto;
    scroll-snap-type: x proximity;
    scrollbar-width: none;
  }
  .card {
    width: clamp(7.5rem, 11vw, 10rem);
    padding: 0;
    border: none;
    background: none;
    cursor: pointer;
    scroll-snap-align: center;
    transition: transform var(--t) var(--ease);
  }
  .see-all {
    aspect-ratio: 3 / 4;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.6rem;
    color: var(--text);
    background: color-mix(in srgb, var(--surface) 80%, transparent);
    border: 3px dashed color-mix(in srgb, var(--text) 40%, transparent);
    font-size: 1rem;
  }
  .card.selected,
  .card:focus {
    transform: translateY(-0.5rem) scale(1.06);
  }
  .badge {
    position: absolute;
    top: -0.5rem;
    right: -0.5rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid #05040a;
    padding: 0.2rem;
    display: grid;
    place-items: center;
  }
  .empty {
    margin: auto;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 36rem;
  }
</style>
