<script lang="ts">
  import { tick } from 'svelte'
  import GameArt from '../components/GameArt.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import HeroVideo from '../components/HeroVideo.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import GameMenu from '../components/GameMenu.svelte'
  import ConsoleIcon from '../components/ConsoleIcon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { toggleFavorite } from '../lib/actions'
  import { downloadGame, downloadText, downloads } from '../lib/downloads.svelte'
  import { indicator } from '../lib/indicator'
  import { shade } from '../lib/motion'
  import { pickRandom, sortGames } from '../lib/library'
  import { sortOf, sourceGames, sourceTitle } from '../lib/sources'
  import { app, currentProfile, go, isKids, shown, system, toast } from '../lib/state.svelte'
  import type { Game, GameCheevos, SaveState } from '../lib/types'
  import { ago, playTime } from '../lib/format'

  // ocultos ficam fora do Início (a página do jogo ainda abre). Duas visões,
  // como o Games/Media do PS5: Biblioteca (o que está neste PC) e Loja (o que está
  // no servidor pra baixar). Perfil infantil não vê a Loja.
  const store = $derived(app.homeView === 'store' && !isKids())
  const remoteCount = $derived(shown().filter((g) => g.remote).length)
  const visible = $derived(shown().filter((g) => (store ? g.remote : !g.remote)))

  function setView(v: 'games' | 'store') {
    if (app.homeView === v) return
    app.homeView = v
    app.homeTab = 'home'
    app.selected = ''
  }

  // abas: só o Início e os consoles. Coleções e listas viram fileiras do Início.
  type Tab = { slug: string; short: string; name: string; kind: 'home' | 'collection' | 'list' | 'system' }
  const tabs = $derived<Tab[]>([
    { slug: 'home', short: 'Início', name: 'Início', kind: 'home' },
    ...app.systems
      .filter((s) => visible.some((g) => g.system === s.slug))
      .map((s) => ({ slug: s.slug, short: s.short, name: s.name, kind: 'system' as const })),
  ])

  type Row = { id: string; title: string; games: Game[]; source?: string; smart?: boolean }
  const rows = $derived.by((): Row[] => {
    if (app.homeTab.startsWith('collection:') || app.homeTab.startsWith('list:')) {
      const games = sortGames(sourceGames(app.homeTab), sortOf(app.homeTab))
      return [{ id: app.homeTab, title: `${sourceTitle(app.homeTab)} · ${games.length} jogo${games.length === 1 ? '' : 's'}`, games, source: app.homeTab }]
    }
    if (app.homeTab !== 'home') {
      const sys = system(app.homeTab)
      const games = sortGames(
        visible.filter((g) => g.system === app.homeTab),
        sortOf(`system:${app.homeTab}`),
      )
      return [{ id: app.homeTab, title: `${sys?.name ?? ''} · ${games.length} jogo${games.length === 1 ? '' : 's'}${store ? ' pra baixar' : ''}`, games, source: `${store ? 'store' : 'system'}:${app.homeTab}` }]
    }
    if (store) {
      // Loja: tudo do servidor e uma fileira por console
      const out: Row[] = [{ id: 'store-all', title: `No servidor · ${visible.length} jogo${visible.length === 1 ? '' : 's'}`, games: sortGames(visible, 'name').slice(0, 30), source: 'store:' }]
      for (const s of app.systems) {
        const games = sortGames(visible.filter((g) => g.system === s.slug), 'name')
        if (games.length) out.push({ id: `store-${s.slug}`, title: `${s.name} · ${games.length}`, games: games.slice(0, 20), source: `store:${s.slug}` })
      }
      return out
    }
    const out: Row[] = []
    const recent = visible.filter((g) => g.lastPlayed > 0).sort((a, b) => b.lastPlayed - a.lastPlayed).slice(0, 12)
    if (recent.length) out.push({ id: 'recent', title: 'Continuar jogando', games: recent, source: 'collection:recent' })
    const favs = visible.filter((g) => g.favorite)
    if (favs.length) out.push({ id: 'favorites', title: 'Favoritos', games: favs, source: 'collection:favorites' })
    for (const l of app.lists) {
      const games = sourceGames(`list:${l.id}`)
      if (games.length) out.push({ id: `list-${l.id}`, title: l.name, games, source: `list:${l.id}`, smart: !!l.rule })
    }
    // a biblioteca toda, sempre por último (ou sozinha, no começo)
    const all = sortGames(visible, sortOf('collection:all'))
    out.push({ id: 'all', title: `Sua biblioteca · ${visible.length} jogo${visible.length === 1 ? '' : 's'}`, games: all.slice(0, 30), source: 'collection:all' })
    return out
  })

  const selected = $derived(visible.find((g) => g.id === app.selected) ?? rows[0]?.games[0])
  const selSystem = $derived(selected ? system(selected.system) : undefined)
  const canResume = $derived(!!selected && selected.lastPlayed > 0 && selSystem?.emulator === 'retroarch')

  // cartões de atividade do destaque: o ponto salvo automático (com a
  // miniatura) e as conquistas. Busca com atraso: o destaque muda a cada passo.
  let autoState = $state<SaveState | null>(null)
  let trophies = $state<GameCheevos | null>(null)
  $effect(() => {
    const g = selected
    autoState = null
    trophies = null
    if (!g || g.remote) return
    const t = setTimeout(() => {
      if (g.hasSaves) api.saves(g.id).then((l) => selected?.id === g.id && (autoState = l.states.find((s) => s.slot === -1) ?? null)).catch(() => {})
      api.gameCheevos(g.id).then((c) => selected?.id === g.id && (trophies = c)).catch(() => {})
    }, 450)
    return () => clearTimeout(t)
  })
  const nextTrophy = $derived(trophies?.list?.find((c) => !c.unlocked))

  // paleta: a do console da aba; no Início, a do jogo selecionado
  $effect(() => {
    if (app.settings?.a11y.highContrast) return applyPalette(highContrastPalette)
    const sys = app.homeTab === 'home' || app.homeTab.includes(':') ? selSystem : system(app.homeTab)
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
    return id ? visible.find((g) => g.id === id) : selected
  }

  let lastRandom = ''
  let spinning = $state(false)

  // Surpresa: roleta rápida pelas capas e para num jogo da aba atual
  async function surprise() {
    if (spinning) return
    const pool = app.homeTab === 'home' ? visible : rows.flatMap((r) => r.games)
    const pick = pickRandom(pool, lastRandom)
    if (!pick) return
    lastRandom = pick.id
    spinning = true
    if (!app.settings?.a11y.reduceMotion) {
      for (let i = 0; i < 8; i++) {
        app.selected = pickRandom(pool, app.selected)?.id ?? pick.id
        play('move')
        await new Promise((r) => setTimeout(r, 70 + i * 12))
      }
    }
    spinning = false
    play('start')
    toast(`Surpresa: ${pick.name}`, 'info', 3000)
    const card = document.querySelector<HTMLElement>(`.shelf [data-game="${pick.id}"]`)
    if (card) {
      app.selected = pick.id
      focusEl(card)
    } else {
      go({ name: 'game', id: pick.id })
    }
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
        case 'menu': {
          // Start num jogo = menu do jogo (como o Options do PS5); fora dele, Você
          const id = current()?.dataset.game
          if (id) menuFor = id
          else go({ name: 'you' })
          return true
        }
        case 'random':
          surprise()
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

  // menu do jogo em foco (Start/Options)
  let menuFor = $state('')

  const hero = $derived(selected?.cover ?? '')
  const me = $derived(currentProfile())
  // placar de fliperama: o "recorde" é o tempo total jogado (HHH:MM)
  const hiScore = $derived.by(() => {
    const total = visible.reduce((sum, g) => sum + (g.playTime || 0), 0)
    const h = Math.floor(total / 3600)
    const m = Math.floor((total % 3600) / 60)
    return `${String(h).padStart(3, '0')}:${String(m).padStart(2, '0')}`
  })
</script>

<div class="home">
  {#key hero}
    <div class="backdrop" in:shade style:background-image={hero ? `url("${hero}")` : 'none'} aria-hidden="true"></div>
  {/key}
  <HeroVideo game={selected} />
  <TopBar>
    {#if !isKids()}
      <nav class="views" aria-label="Visão">
        <button class="view pixel" data-nav data-view="games" aria-current={!store ? 'page' : undefined} onclick={() => setView('games')}>Biblioteca</button>
        <button class="view pixel" data-nav data-view="store" aria-current={store ? 'page' : undefined} onclick={() => setView('store')}>
          Loja{#if remoteCount}<span class="count">{remoteCount}</span>{/if}
        </button>
      </nav>
    {/if}
  </TopBar>
  <!-- consoles como no PS5: blocos com o desenho de cada um; o escolhido cresce e mostra o nome -->
  <nav class="tabs" aria-label="Consoles" use:indicator={app.homeTab}>
    {#each tabs as t (t.slug)}
      <button
        class="tab"
        class:first-system={t.kind === 'system' && tabs.find((x) => x.kind === 'system') === t}
        data-nav
        data-tab={t.slug}
        data-nav-group="tabs"
        data-nav-default={t.slug === app.homeTab ? '' : undefined}
        aria-current={t.slug === app.homeTab ? 'page' : undefined}
        onfocus={() => selectTab(t.slug)}
        onclick={() => selectTab(t.slug)}
      >
        <span class="sr-only">{t.short}{#if t.name !== t.short} ({t.name}){/if}</span>
        <ConsoleIcon slug={t.slug} size={t.slug === app.homeTab ? '4.2rem' : '3rem'} />
        {#if t.slug === app.homeTab}<span class="caption" aria-hidden="true">{t.name}</span>{/if}
      </button>
    {/each}
  </nav>

  <main class="content" aria-labelledby="hero-title">
    {#if visible.length}
      <div class="hud pixel" aria-label="Placar">
        <p><span>1UP</span> {(me?.name ?? 'Jogador').toUpperCase()}</p>
        <p><span>HI-SCORE</span> {hiScore}</p>
        <p><span>JOGOS</span> {String(visible.length).padStart(3, '0')}</p>
      </div>
    {/if}
    {#if visible.length === 0 && store}
      <section class="empty panel">
        <h1 class="pixel" id="hero-title">Loja vazia</h1>
        <p class="muted">A Loja mostra os jogos do seu servidor RomM que ainda não estão neste PC. Conecte o servidor (ou já baixou tudo).</p>
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'you', item: 'server' })}><Icon name="cloud" /> Servidor RomM</button>
          <button class="btn" data-nav onclick={() => setView('games')}><Icon name="pad" /> Ir pra Biblioteca</button>
        </div>
      </section>
    {:else if visible.length === 0}
      <section class="empty panel">
        <h1 class="pixel" id="hero-title">Nenhum jogo ainda</h1>
        <p class="muted">{remoteCount ? `Tem ${remoteCount} jogo${remoteCount === 1 ? '' : 's'} no servidor esperando: baixe pela Loja.` : 'Importe ROMs ou aponte suas pastas de jogos nas configurações.'}</p>
        <div class="actions">
          {#if remoteCount && !isKids()}<button class="btn primary" data-nav data-nav-default onclick={() => setView('store')}><Icon name="download" /> Abrir a Loja</button>{/if}
          <button class="btn" class:primary={!remoteCount} data-nav data-nav-default={remoteCount ? undefined : ''} onclick={() => go({ name: 'import' })}><Icon name="plus" /> Importar jogos</button>
          <button class="btn" data-nav onclick={() => go({ name: 'you' })}><Icon name="gear" /> Configurações</button>
        </div>
      </section>
    {:else if selected}
      <div class="stage">
      <section class="hero">
        <p class="system-pill pixel">{selSystem?.name ?? selected.system}</p>
        {#key selected.id}
          <!-- título digitado letra por letra (um degrau por letra) -->
          <h1 class="pixel title typing" id="hero-title" style:--letters={selected.name.length} style:animation-timing-function={`steps(${Math.max(1, selected.name.length)})`}>{selected.name}</h1>
        {/key}
        <p class="meta">
          {#if selected.year}<span>{selected.year}</span>{/if}
          {#if selected.players}<span>{selected.players} jogador{selected.players === '1' ? '' : 'es'}</span>{/if}
          {#each selected.genres.slice(0, 2) as genre (genre)}<span>{genre}</span>{/each}
          {#if selected.favorite}<span class="fav"><Icon name="star" /> Favorito</span>{/if}
        </p>
        {#if selected.summary}<p class="summary">{selected.summary}</p>{/if}
        <div class="actions">
          {#if selected.remote}
            <button class="btn primary" data-nav data-nav-group="actions" disabled={downloads[selected.id] !== undefined || isKids()} onclick={() => downloadGame(selected)}>
              <Icon name="download" />
              {downloads[selected.id] !== undefined ? downloadText(selected.id) : 'Baixar do servidor'}
            </button>
          {:else}
            <button class="btn primary" data-nav data-nav-group="actions" onclick={() => launch(selected)}>
              <Icon name="play" /> Jogar
            </button>
          {/if}
          {#if canResume && !selected.remote && !autoState}
            <button class="btn" data-nav data-nav-group="actions" onclick={() => launch(selected, 'auto')}>
              <Icon name="resume" /> Continuar
            </button>
          {/if}
          <button class="btn" data-nav data-nav-group="actions" onclick={() => go({ name: 'game', id: selected.id })}>
            <Icon name="disc" /> Detalhes
          </button>
        </div>
        <!-- espaço fixo: o destaque troca a cada passo na fileira e as fileiras não podem pular -->
        <div class="cards" data-nav-group="cards">
            {#if autoState && canResume}
              <button class="card-act" data-nav onclick={() => launch(selected, 'auto')}>
                {#if autoState.thumb}<img src={`/media/thumb/${autoState.id}`} alt="" />{:else}<span class="nothumb pixel" aria-hidden="true">AUTO</span>{/if}
                <span class="text">
                  <span class="label">Continuar de onde parou</span>
                  <span class="muted">salvo {ago(autoState.modified)} · {playTime(selected.playTime)} jogados</span>
                </span>
              </button>
            {/if}
            {#if trophies?.supported && trophies.total}
              <button class="card-act" data-nav onclick={() => go({ name: 'game', id: selected.id })}>
                <span class="trophy"><Icon name="trophy" size="1.8em" /></span>
                <span class="text">
                  <span class="label">{trophies.unlocked}/{trophies.total} conquistas</span>
                  <span class="muted">{nextTrophy ? `próxima: ${nextTrophy.title}` : 'todas ganhas!'}</span>
                </span>
              </button>
            {/if}
        </div>
      </section>
      <div class="hero-art" aria-hidden="true">
        {#key selected.id}<GameArt game={selected} system={selSystem} large />{/key}
      </div>
      </div>

      <div class="shelves">
        {#each rows as row (row.id)}
          <section class="shelf" aria-label={row.title}>
            <h2 class="row-title">
              {row.title}{#if row.smart}<span class="smart" title="Lista automática"><Icon name="auto" size="0.8em" /><span class="sr-only">(lista automática)</span></span>{/if}
            </h2>
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
                    aria-label={`${g.name}, ${sys?.short ?? g.system}${g.year ? `, ${g.year}` : ''}${g.favorite ? ', favorito' : ''}${g.remote ? ', no servidor' : ''}`}
                    onfocus={() => (app.selected = g.id)}
                    onclick={() => go({ name: 'game', id: g.id })}
                  >
                    <GameArt game={g} system={sys} />
                    {#if g.favorite}<span class="badge" aria-hidden="true"><Icon name="star" /></span>{/if}
                    {#if g.remote}<span class="badge cloud" aria-hidden="true"><Icon name="cloud" /></span>{/if}
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
    coin
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'north', label: 'Favorito' },
      { button: 'west', label: 'Detalhes' },
      { button: 'l1', label: 'Console anterior' },
      { button: 'r1', label: 'Próximo console' },
      { button: 'select', label: 'Buscar' },
      { button: 'r3', label: 'Surpresa' },
      { button: 'start', label: 'Opções do jogo · Você' },
    ]}
  />
</div>
{#if menuFor}<GameMenu gameId={menuFor} onclose={() => (menuFor = '')} />{/if}

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
  .views {
    display: flex;
    gap: 0.4rem;
  }
  .view {
    font-size: 1.15rem;
    color: var(--muted);
    background: none;
    border: 3px solid transparent;
    padding: 0.4rem 0.9rem;
    cursor: pointer;
    display: inline-flex;
    gap: 0.5rem;
    align-items: center;
  }
  .view[aria-current='page'] {
    color: #fff;
    text-shadow: 0 0 10px var(--accent);
  }
  .view .count {
    font-size: 0.7rem;
    color: var(--on-accent);
    background: var(--accent);
    padding: 0.15rem 0.35rem;
  }
  .tabs {
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: 0.9rem;
    overflow-x: auto;
    scrollbar-width: none;
    padding: 0.4rem 2.5rem 2.2rem;
    min-height: 9.4rem;
  }
  /* luz do console escolhido: desliza entre os blocos (lib/indicator.ts) */
  .tabs :global(.tab-light) {
    position: absolute;
    left: 0;
    top: 0;
    border: 3px solid var(--accent);
    box-shadow: var(--glow);
    transition:
      transform 200ms steps(5),
      width 200ms steps(5),
      height 200ms steps(5),
      border-color 300ms;
    pointer-events: none;
  }
  .typing {
    animation: typing calc(var(--letters) * 28ms + 60ms) both;
  }
  @keyframes typing {
    from {
      clip-path: inset(0 100% 0 0);
    }
    to {
      clip-path: inset(0 -1em 0 0);
    }
  }
  .tab {
    position: relative;
    z-index: 1;
    flex: none;
    display: grid;
    place-items: center;
    width: 5rem;
    height: 5rem;
    background: color-mix(in srgb, var(--surface) 75%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
    cursor: pointer;
  }
  .tab[aria-current='page'] {
    width: 6.8rem;
    height: 6.8rem;
    background: color-mix(in srgb, var(--accent) 22%, var(--surface));
    border-color: transparent;
  }
  /* tela baixa (notebook): faixa dos consoles mais compacta */
  @media (max-height: 820px) {
    /* (seletor mais forte: as regras normais vêm depois no arquivo) */
    .content .hud {
      display: none;
    }
    .hero .cards {
      min-height: 0;
    }
    .tabs {
      min-height: 7.4rem;
      padding-bottom: 1.8rem;
    }
    .tab {
      width: 4rem;
      height: 4rem;
    }
    .tab[aria-current='page'] {
      width: 5.2rem;
      height: 5.2rem;
    }
  }
  .tab.first-system {
    margin-left: 0.8rem;
  }
  .caption {
    position: absolute;
    top: calc(100% + 0.5rem);
    left: 0;
    white-space: nowrap;
    font-size: 1.15rem;
    color: #fff;
    text-shadow: 0 0 8px var(--accent);
  }
  .hud {
    display: flex;
    justify-content: space-between;
    gap: 2rem;
    padding: 0.2rem 0 0.8rem;
    font-size: 1.2rem;
    color: #fff;
  }
  .hud p {
    line-height: 1;
  }
  .hud span {
    margin-right: 0.6em;
    color: var(--accent);
    text-shadow: 0 0 8px var(--accent);
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
    /* nunca encolhe abaixo do destaque: com várias fileiras, a tela rola
       (o foco leva a fileira até a vista) em vez de uma cobrir a outra */
    flex: 1 0 auto;
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
  @media (max-height: 820px) {
    .hero-art {
      display: none;
    }
  }
  .hero {
    max-width: 52rem;
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  .system-pill {
    color: var(--accent-2);
    font-size: 1rem;
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  /* nome do jogo como o logo no destaque do PS5: bem grande */
  .title {
    font-size: clamp(2.2rem, min(6.5vw, 8vh), 6rem);
    line-height: 1;
    color: #fff;
    text-shadow:
      0 0 14px var(--accent),
      4px 4px 0 color-mix(in srgb, var(--accent) 70%, #000);
    max-width: 18ch;
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
  .cards {
    min-height: 4.8rem;
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
  }
  .card-act {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 0.8rem;
    align-items: center;
    max-width: 22rem;
    padding: 0.5rem 0.9rem 0.5rem 0.5rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--surface) 80%, transparent);
    border: 2px solid color-mix(in srgb, var(--accent-2) 50%, transparent);
    cursor: pointer;
  }
  .card-act img,
  .card-act .nothumb,
  .card-act .trophy {
    width: 5.5rem;
    height: 3.6rem;
    object-fit: cover;
    image-rendering: pixelated;
    display: grid;
    place-items: center;
    background: #000;
    color: var(--focus);
  }
  .card-act .text {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
  }
  .card-act .label {
    font-weight: 700;
  }
  .card-act .muted {
    font-size: 0.85rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .shelves {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    padding-bottom: 0.5rem;
  }
  .row-title .smart {
    margin-left: 0.5rem;
    opacity: 0.7;
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
    /* capa grande (é o que a pessoa procura), mas cabendo na altura da TV */
    width: clamp(8.5rem, min(14.5vw, 22vh), 14rem);
    padding: 0;
    border: none;
    background: none;
    /* contorno neon da cor do console (o foco troca por amarelo) */
    box-shadow:
      0 0 0 2px color-mix(in srgb, var(--accent) 75%, transparent),
      0 0 14px color-mix(in srgb, var(--accent) 35%, transparent);
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
  /* foco: a borda acende piscando como letreiro e depois "respira" */
  .card:focus {
    animation:
      neon-on 0.24s steps(1) both,
      breathe 1.6s ease-in-out 0.24s infinite;
  }
  /* sombra de luz no "chão" debaixo da capa em foco */
  .card:focus::after {
    content: '';
    position: absolute;
    left: 8%;
    right: 8%;
    bottom: -1.1rem;
    height: 0.6rem;
    background: radial-gradient(ellipse at center, color-mix(in srgb, var(--focus) 45%, transparent), transparent 70%);
    pointer-events: none;
  }
  @keyframes neon-on {
    0%,
    50% {
      outline-color: transparent;
      box-shadow: none;
    }
    25%,
    75% {
      outline-color: var(--focus);
      box-shadow: var(--glow-focus);
    }
  }
  @keyframes breathe {
    0%,
    100% {
      box-shadow: 0 0 8px var(--focus), 0 0 20px color-mix(in srgb, var(--focus) 40%, transparent);
    }
    50% {
      box-shadow: 0 0 14px var(--focus), 0 0 40px color-mix(in srgb, var(--focus) 65%, transparent);
    }
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
  .badge.cloud {
    right: auto;
    left: -0.5rem;
    background: var(--accent-2);
  }
  .empty {
    margin: auto;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    max-width: 36rem;
  }
</style>
