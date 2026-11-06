<script lang="ts">
  import { tick } from 'svelte'
  import Cheevos from '../components/Cheevos.svelte'
  import GameArt from '../components/GameArt.svelte'
  import GameConfig from '../components/GameConfig.svelte'
  import GameOptions from '../components/GameOptions.svelte'
  import ListPicker from '../components/ListPicker.svelte'
  import MediaViewer from '../components/MediaViewer.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api, on } from '../lib/api'
  import { ago, bytes, dateTime, playTime } from '../lib/format'
  import { downloadGame, downloadText, downloads } from '../lib/downloads.svelte'
  import { pushHandler } from '../lib/input'
  import { burst } from '../lib/motion'
  import { current, focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, back, confirm, game, isKids, prompt, putGame, system, toast } from '../lib/state.svelte'
  import type { BiosInfo, GameEvent, KidsTimeInfo, Listing, MediaFile, MediaItem, SaveState } from '../lib/types'

  let { id }: { id: string } = $props()
  const g = $derived(game(id))
  const sys = $derived(g ? system(g.system) : undefined)
  let listing = $state<Listing | null>(null)
  let shots = $state<MediaFile[]>([])
  let note = $state('')
  let loadError = $state('')
  let stamp = $state(Date.now())
  let picking = $state(false)
  let menu = $state(false)
  let configuring = $state(false)
  let optionsBtn = $state<HTMLButtonElement>()
  let viewing = $state<number | null>(null)
  const media = $derived.by(() => {
    if (!g) return [] as MediaItem[]
    const out: MediaItem[] = []
    if (g.cover) out.push({ src: g.cover, caption: 'Capa', kind: 'image' })
    if (g.snap) out.push({ src: g.snap, caption: 'Tela do jogo', kind: 'image' })
    if (g.titleShot) out.push({ src: g.titleShot, caption: 'Tela-título', kind: 'image' })
    for (const st of listing?.states ?? []) {
      if (st.thumb) out.push({ src: `/media/thumb/${st.id}?v=${stamp}`, caption: `Ponto salvo: ${st.label}`, kind: 'image' })
    }
    for (const m of shots) {
      out.push({ src: `/media/shot/${m.id}`, caption: `${m.kind === 'video' ? 'Gravação' : 'Captura'} de ${dateTime(m.at)}`, kind: m.kind })
    }
    if (g.manual) out.push({ src: g.manual, caption: 'Manual', kind: 'pdf' })
    return out
  })
  // BIOS obrigatória do console faltando: um aviso só, acima dos botões
  let bios = $state<BiosInfo | null>(null)
  const orList = (xs: string[]) => (xs.length > 1 ? `${xs.slice(0, -1).join(', ')} ou ${xs[xs.length - 1]}` : (xs[0] ?? ''))
  $effect(() => {
    const slug = g?.system
    if (!slug) return
    api
      .biosStatus()
      .then((all) => (bios = all.find((b) => b.system === slug && b.required && !b.ok && !b.fallback) ?? null))
      .catch(() => (bios = null))
  })
  // perfil infantil com limite: sem tempo, o Jogar fica desligado com o motivo
  let kidsTime = $state<KidsTimeInfo | null>(null)
  const blocked = $derived(kidsTime?.reason ?? '')
  const loadKidsTime = () => {
    if (!isKids()) return
    api
      .kidsTime()
      .then((t) => {
        kidsTime = t
        if (t.reason) tick().then(focusDefault) // o Jogar desligou: foco vai pro próximo
      })
      .catch(() => {})
  }

  async function load() {
    try {
      listing = await api.saves(id)
      shots = await api.gameMedia(id).catch(() => [])
      note = await api.gameNote(id).catch(() => '')
      loadError = ''
      stamp = Date.now()
    } catch (err) {
      loadError = (err as Error).message
    }
  }

  $effect(() => {
    load()
    loadKidsTime()
    tick().then(focusDefault)
    // depois de jogar, recarrega (state novo, miniatura nova)
    return on<GameEvent>('game', (e) => {
      if (e.state === 'exited' && e.id === id) load()
      if (e.state === 'exited') loadKidsTime()
    })
  })

  $effect(() => {
    if (app.settings?.a11y.highContrast) applyPalette(highContrastPalette)
    else applyPalette(sys?.palette ?? homePalette)
  })

  const auto = $derived(listing?.states.find((s) => s.slot === -1))

  async function launch(state = '') {
    if (!g) return
    try {
      play('start')
      await api.play(g.id, state)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  async function editNote() {
    await prompt('Anotação deste jogo', note, 'Salvar', async (v) => {
      try {
        await api.setGameNote(id, v)
        note = v.trim()
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 2000, multiline: true })
  }

  async function toggleFav() {
    if (!g) return
    try {
      const from = current()
      const updated = await api.setFavorite(g.id, !g.favorite)
      putGame(updated)
      play('toggle')
      if (updated.favorite) burst(from)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function remove(st: SaveState) {
    const ok = await confirm(`Apagar o ponto salvo "${st.label}" de ${dateTime(st.modified)}? Não dá pra desfazer.`, 'Apagar', true)
    if (!ok) return
    try {
      listing = await api.deleteState(id, st.id)
      toast('Ponto salvo apagado', 'ok', 2500)
      tick().then(focusDefault)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() =>
    pushHandler((a) => {
      if (menu || picking || configuring || viewing !== null) return false
      if (a === 'fav') {
        const stateId = current()?.dataset.state
        const st = isKids() ? undefined : listing?.states.find((s) => s.id === stateId)
        if (st) remove(st)
        else toggleFav()
        return true
      }
      return a === 'details' || a === 'prevTab' || a === 'nextTab'
    }),
  )
</script>

{#if g}
  <div class="page">
    <div class="backdrop" style:background-image={g.cover ? `url("${g.cover}")` : 'none'} aria-hidden="true"></div>
    <TopBar />
    <main class="layout">
      <div class="art">
        <GameArt game={g} system={sys} large />
      </div>
      <div class="info">
        <p class="system pixel">{sys?.name ?? g.system}</p>
        <h1 class="pixel">{g.name}</h1>
        <ul class="facts">
          {#if g.remote}<li class="chip remote">No servidor</li>{/if}
          {#if g.hidden}<li class="chip">Oculto</li>{/if}
          {#if g.rating}<li class="chip rating" aria-label={`Nota ${g.rating} de 5`}>{'★'.repeat(g.rating)}</li>{/if}
          {#if g.completed}<li class="chip done">Completo</li>{/if}
          {#if g.broken}<li class="chip broken">Quebrado</li>{/if}
          {#if g.year}<li class="chip">{g.year}</li>{/if}
          {#if g.developer}<li class="chip">{g.developer}</li>{/if}
          {#if g.players}<li class="chip">{g.players} jogador{g.players === '1' ? '' : 'es'}</li>{/if}
          {#each g.genres as genre (genre)}<li class="chip">{genre}</li>{/each}
          <li class="chip">{g.plays} partida{g.plays === 1 ? '' : 's'}</li>
          <li class="chip">Tempo: {playTime(g.playTime)}</li>
          <li class="chip">Última vez: {ago(g.lastPlayed)}</li>
          <li class="chip">{bytes(g.size)}</li>
        </ul>
        {#if g.summary}<p class="summary">{g.summary}</p>{/if}
        {#if bios}
          <p class="bios-warn" role="note">
            <Icon name="warn" />
            <span>Falta a BIOS do {bios.name} ({orList(bios.missing.map((m) => m.file))}). Suba no seu RomM (página do console › Firmware) que o app baixa sozinho, ou coloque em <span class="path">{bios.dir}</span></span>
          </p>
        {/if}
        <div class="actions">
          {#if g.remote}
            {#if downloads[g.id] !== undefined}
              <div class="downloading" role="status">
                <span><Icon name="download" /> {downloadText(g.id)}</span>
                <span class="energy" aria-hidden="true"><span style:width={`${Math.floor(downloads[g.id] * 20) * 5}%`}></span></span>
              </div>
            {:else if isKids()}
              <p class="muted">Este jogo está no servidor. Peça pra um adulto baixar.</p>
            {:else}
              <button class="btn primary" data-nav data-nav-default onclick={() => downloadGame(g)}>
                <Icon name="download" /> Baixar do servidor ({bytes(g.size)})
              </button>
            {/if}
          {:else}
            <button class="btn primary" data-nav data-nav-default={!blocked || undefined} disabled={!!blocked} aria-describedby={blocked ? 'kids-blocked' : undefined} onclick={() => launch()}>
              <Icon name="play" /> Jogar do início
            </button>
          {/if}
          {#if auto}
            <button class="btn" data-nav disabled={!!blocked} onclick={() => launch('auto')}><Icon name="resume" /> Continuar de onde parei</button>
          {/if}
          <button class="btn" data-nav data-nav-default={blocked ? true : undefined} aria-pressed={g.favorite} onclick={toggleFav}>
            <Icon name="star" /> {g.favorite ? 'Favorito' : 'Favoritar'}
          </button>
          <button class="btn" data-nav aria-haspopup="dialog" bind:this={optionsBtn} onclick={() => (menu = true)}>
            <Icon name="sliders" /> Opções
          </button>
        </div>
        {#if blocked}<p class="blocked" id="kids-blocked" role="status">{blocked}</p>{/if}

        {#if media.length}
          <section class="media" aria-labelledby="media-title">
            <h2 id="media-title">Mídia</h2>
            <ul class="strip">
              {#each media as m, i (m.src)}
                <li>
                  <button class="thumb" data-nav aria-haspopup="dialog" onclick={() => (viewing = i)}>
                    {#if m.kind === 'pdf'}
                      <span class="nothumb pixel" aria-hidden="true">PDF</span>
                    {:else if m.kind === 'video'}
                      <span class="nothumb pixel" aria-hidden="true">VÍDEO</span>
                    {:else}
                      <img src={m.src} alt="" loading="lazy" />
                    {/if}
                    <span class="thumb-label"><span class="sr-only">Ver </span>{m.caption}</span>
                  </button>
                </li>
              {/each}
            </ul>
          </section>
        {/if}

        {#if !g.remote}<Cheevos {id} />{/if}

        {#if note}
          <section class="note-box" aria-labelledby="note-title">
            <h2 id="note-title">Sua anotação</h2>
            <p class="note-text">{note}</p>
            <button class="btn small" data-nav onclick={editNote}><Icon name="pencil" /> Editar anotação</button>
          </section>
        {/if}

        <section class="saves" aria-labelledby="saves-title">
          <h2 id="saves-title">Pontos salvos</h2>
          {#if loadError}
            <p class="error">{loadError}</p>
          {:else if listing}
            {#if listing.states.length === 0}
              <p class="muted">Nenhum ponto salvo ainda. No jogo, segure Select e aperte R pra salvar.</p>
            {:else}
              <ul class="states">
                {#each listing.states as st (st.id)}
                  <li>
                    <button
                      class="state"
                      data-nav
                      data-state={st.id}
                      title={`Salvo em ${dateTime(st.modified)}`}
                      onclick={() => launch(st.id)}
                    >
                      {#if st.thumb}
                        <img src={`/media/thumb/${st.id}?v=${stamp}`} alt="" loading="lazy" />
                      {:else}
                        <span class="nothumb pixel" aria-hidden="true">{st.slot === -1 ? 'AUTO' : `#${st.slot}`}</span>
                      {/if}
                      <span class="state-label"><span class="sr-only">Carregar </span>{st.label}</span>
                      <span class="state-date muted">{ago(st.modified)}</span>
                    </button>
                  </li>
                {/each}
              </ul>
            {/if}
            {#if listing.files.length}
              <h3 class="sub">Saves do jogo</h3>
              <ul class="files">
                {#each listing.files as f (f.path)}
                  <li>
                    <Icon name="folder" />
                    <span>{f.name}</span>
                    <span class="muted">{bytes(f.size)} · {ago(f.modified)}{f.shared ? ' · compartilhado entre jogos' : ''}</span>
                  </li>
                {/each}
              </ul>
            {/if}
            {#if listing.note}<p class="muted note">{listing.note}</p>{/if}
          {/if}
        </section>
      </div>
    </main>
    <HelpBar
      hints={[
        { button: 'south', label: 'Escolher' },
        { button: 'north', label: isKids() ? 'Favorito' : 'Favorito / apagar save' },
        { button: 'east', label: 'Voltar' },
      ]}
    />
  </div>
  {#if menu}
    <GameOptions
      gameId={id}
      onclose={() => {
        menu = false
        tick().then(() => optionsBtn?.focus())
      }}
      onlists={() => {
        menu = false
        picking = true
      }}
      onconfig={() => {
        menu = false
        configuring = true
      }}
    />
  {/if}
  {#if configuring}
    <GameConfig
      gameId={id}
      onclose={() => {
        configuring = false
        tick().then(() => optionsBtn?.focus())
      }}
    />
  {/if}
  {#if picking}
    <ListPicker
      gameId={id}
      onclose={() => {
        picking = false
        tick().then(() => optionsBtn?.focus())
      }}
    />
  {/if}
  {#if viewing !== null && media.length}
    <MediaViewer
      items={media}
      start={Math.min(viewing, media.length - 1)}
      gameId={id}
      onclose={() => {
        viewing = null
        tick().then(focusDefault)
      }}
    />
  {/if}
{:else}
  <main class="layout">
    <p>Jogo não encontrado.</p>
    <button class="btn" data-nav data-nav-default onclick={() => back()}>Voltar</button>
  </main>
{/if}

<style>
  .note-box {
    display: grid;
    gap: 0.5rem;
    justify-items: start;
  }
  .note-text {
    margin: 0;
    white-space: pre-line;
    padding: 0.7rem 0.9rem;
    border-left: 4px solid var(--focus);
    background: color-mix(in srgb, var(--surface) 80%, transparent);
  }
  .bios-warn {
    display: flex;
    gap: 0.6rem;
    align-items: flex-start;
    margin: 0;
    padding: 0.5rem 0.8rem;
    border-left: 3px solid var(--focus);
    font-size: 0.9rem;
  }
  .bios-warn :global(svg) {
    color: var(--focus);
    flex: none;
    margin-top: 0.2rem;
  }
  .bios-warn .path {
    overflow-wrap: anywhere;
    font-family: monospace;
  }
  .blocked {
    font-weight: 700;
    color: var(--focus);
  }
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .backdrop {
    position: fixed;
    inset: 0;
    background-size: cover;
    background-position: center;
    filter: blur(14px) saturate(1.2);
    opacity: 0.22;
    z-index: -1;
  }
  .layout {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: grid;
    grid-template-columns: minmax(12rem, 22rem) 1fr;
    gap: 3rem;
    padding: 1rem 2.5rem 2rem 3.5rem;
    align-items: start;
  }
  .info {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    min-width: 0;
  }
  .system {
    color: var(--accent-2);
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }
  h1 {
    font-size: clamp(2rem, 4vw, 3.2rem);
  }
  .facts {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .chip.rating,
  .chip.done {
    color: var(--accent);
    border-color: currentColor;
  }
  .chip.remote {
    color: var(--accent-2);
    border-color: currentColor;
  }
  .downloading {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    font-weight: 700;
    color: var(--accent);
  }
  .chip.broken {
    color: #ffb4b4;
    border-color: currentColor;
  }
  .summary {
    max-width: 48rem;
  }
  .actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
  }
  .media h2 {
    font-size: 1.4rem;
  }
  .strip {
    list-style: none;
    padding: 0.4rem;
    margin: 0.6rem 0 0;
    display: flex;
    gap: 1rem;
    overflow-x: auto;
  }
  .thumb {
    width: 10rem;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.4rem;
    font: inherit;
    color: var(--text);
    background: #000;
    border: 2px solid color-mix(in srgb, var(--accent-2) 70%, transparent);
    box-shadow: var(--glow-2);
    cursor: pointer;
  }
  .thumb img,
  .thumb .nothumb {
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    image-rendering: pixelated;
    background: #000;
  }
  .thumb-label {
    font-weight: 700;
    text-align: left;
  }
  .saves {
    margin-top: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  .saves h2 {
    font-size: 1.4rem;
  }
  .sub {
    font-size: 1.05rem;
    margin-top: 0.5rem;
  }
  .states {
    list-style: none;
    padding: 0.4rem;
    margin: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(12rem, 1fr));
    gap: 1rem;
  }
  .state {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.5rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: #000;
    border: 2px solid color-mix(in srgb, var(--accent-2) 70%, transparent);
    box-shadow: var(--glow-2);
    cursor: pointer;
  }
  .state img,
  .nothumb {
    width: 100%;
    aspect-ratio: 4 / 3;
    object-fit: cover;
    image-rendering: pixelated;
    background: #000;
  }
  .nothumb {
    display: grid;
    place-items: center;
    color: var(--accent);
    font-size: 1.6rem;
  }
  .state-label {
    font-weight: 700;
  }
  .files {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.4rem;
  }
  .files li {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .error {
    color: #ff8a8a;
  }
  .note {
    font-style: italic;
  }
</style>
