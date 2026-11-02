<script lang="ts">
  // "Você": perfil e configurações num lugar só, como as do PS5. Lista à
  // esquerda (só pra você / deste console), conteúdo do item à direita, na
  // mesma tela. Voltar: do conteúdo pra lista, da lista pro Início. Itens que
  // são telas inteiras (estatísticas, central...) abrem e voltam pra cá, no
  // mesmo item.
  import { tick, untrack, type Component } from 'svelte'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { play } from '../lib/sound'
  import A11yPanel from './settings/A11yPanel.svelte'
  import AboutPanel from './settings/AboutPanel.svelte'
  import BluetoothPanel from './settings/BluetoothPanel.svelte'
  import CheevosPanel from './settings/CheevosPanel.svelte'
  import ControlsPanel from './settings/ControlsPanel.svelte'
  import EmulatorsPanel from './settings/EmulatorsPanel.svelte'
  import LibraryPanel from './settings/LibraryPanel.svelte'
  import ServerPanel from './settings/ServerPanel.svelte'
  import StoragePanel from './settings/StoragePanel.svelte'
  import UpdatesPanel from './settings/UpdatesPanel.svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { activity, unread } from '../lib/activity.svelte'
  import { api } from '../lib/api'
  import { avatarName } from '../lib/avatars'
  import { summarize, type ProfileSummary } from '../lib/profilestats'
  import { bytes, playTime } from '../lib/format'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app, currentProfile, go, isKids, system, type Screen } from '../lib/state.svelte'
  import type { BTInfo, CheevosInfo, EmulatorsInfo, KidsTimeInfo, RemoteInfo, StorageInfo } from '../lib/types'
  import { versionArrow } from '../lib/update'

  const me = $derived(currentProfile())
  const kids = $derived(isKids())
  let kidsTime = $state<KidsTimeInfo | null>(null)
  $effect(() => {
    if (kids) api.kidsTime().then((t) => (kidsTime = t)).catch(() => {})
  })
  let remote = $state<RemoteInfo | null>(null)
  let storage = $state<StorageInfo | null>(null)
  let emus = $state<EmulatorsInfo | null>(null)
  let summary = $state<ProfileSummary | null>(null)
  let blue = $state<BTInfo | null>(null)
  let cheevos = $state<CheevosInfo | null>(null)

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    api.stats('all').then((s) => (summary = summarize(s))).catch(() => {})
    if (isKids()) return
    api.remoteStatus().then((r) => (remote = r)).catch(() => {})
    api.storage().then((s) => (storage = s)).catch(() => {})
    api.emulatorStatus().then((e) => (emus = e)).catch(() => {})
    api.btStatus().then((b) => (blue = b)).catch(() => {})
    api.cheevosStatus().then((c) => (cheevos = c)).catch(() => {})
  })

  const hours = $derived(Math.floor(app.games.reduce((t, g) => t + (g.playTime || 0), 0) / 3600))
  const played = $derived(app.games.filter((g) => g.lastPlayed > 0).length)
  const favs = $derived(app.games.filter((g) => g.favorite).length)

  function emuSummary(): string {
    if (!emus) return '…'
    const miss = emus.emulators.filter((e) => !e.installed).length + emus.cores.filter((c) => !c.installed).length
    return miss ? `faltam ${miss} (instalar)` : 'tudo instalado'
  }

  let { item = '' }: { item?: string } = $props()

  // painel = abre aqui do lado; tela = abre por cima e volta pra cá; ação = faz na hora
  type Item = { id: string; label: string; icon: IconName; value: string; panel?: Component; screen?: Screen; action?: () => void }
  const to = (s: Screen) => ({ screen: s })
  const panel = (p: Component) => ({ panel: p })
  const PANELS: Record<string, Component> = {
    controls: ControlsPanel,
    a11y: A11yPanel,
    cheevos: CheevosPanel,
    library: LibraryPanel,
    server: ServerPanel,
    storage: StoragePanel,
    emulators: EmulatorsPanel,
    bluetooth: BluetoothPanel,
    updates: UpdatesPanel,
    about: AboutPanel,
  }

  const mine = $derived.by((): Item[] => {
    const a = app.settings?.a11y
    const access = a
      ? [`texto ${Math.round(a.fontScale * 100)}%`, a.highContrast && 'alto contraste', a.reduceMotion && 'sem animações', !a.sounds && 'sem sons']
          .filter(Boolean)
          .join(' · ')
      : ''
    const pad = app.pads.length ? app.pads.map((p) => p.name).join(', ') : 'nenhum controle conectado'
    const news = unread()
    const busy = activity.tasks.filter((t) => t.state === 'running').length
    const items: Item[] = [
      { id: 'activity', label: 'Central', icon: 'bell', value: busy ? `${busy} em andamento` : news ? `${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : 'downloads e avisos', ...to({ name: 'activity' }) },
      { id: 'stats', label: 'Estatísticas', icon: 'chart', value: `${hours} h jogadas · ${played} jogos`, ...to({ name: 'stats' }) },
    ]
    if (!kids) {
      items.splice(1, 0, { id: 'profile', label: 'Perfil', icon: 'pencil', value: `${me?.name ?? ''} · ${avatarName(me?.avatar ?? '')}${me?.hasPIN ? ' · com PIN' : ''}`, ...to({ name: 'profile', id: me?.id }) })
      items.push(
        { id: 'controls', label: 'Controles', icon: 'pad', value: pad, ...panel(PANELS.controls) },
        { id: 'a11y', label: 'Acessibilidade e som', icon: 'sliders', value: access, ...panel(PANELS.a11y) },
        { id: 'cheevos', label: 'Conquistas', icon: 'trophy', value: cheevos ? (cheevos.connected ? `RetroAchievements: ${cheevos.user}` : 'desligadas') : '…', ...panel(PANELS.cheevos) },
      )
    }
    return items
  })

  const house = $derived.by((): Item[] => {
    if (kids) return []
    const s = app.settings
    const saver = s?.screensaver
    const items: Item[] = [
      { id: 'library', label: 'Biblioteca', icon: 'folder', value: `${s?.libraryDirs.length ?? 0} pasta${s?.libraryDirs.length === 1 ? '' : 's'} · ${app.games.length} jogos · capas ${s?.covers.enabled ? 'ligadas' : 'desligadas'}`, ...panel(PANELS.library) },
      { id: 'import', label: 'Importar jogos', icon: 'plus', value: 'de qualquer pasta ou pendrive', ...to({ name: 'import' }) },
      { id: 'server', label: 'Servidor RomM', icon: 'cloud', value: remote ? (remote.connected ? `conectado · ${remote.games} jogos` : 'offline') : '…', ...panel(PANELS.server) },
      { id: 'storage', label: 'Armazenamento', icon: 'disc', value: storage ? (storage.total > 0 ? `${bytes(storage.free)} livres de ${bytes(storage.total)}` : 'jogos: ' + bytes(storage.games)) : '…', ...panel(PANELS.storage) },
      { id: 'emulators', label: 'Emuladores', icon: 'pad', value: emuSummary(), ...panel(PANELS.emulators) },
      { id: 'bluetooth', label: 'Bluetooth', icon: 'bluetooth', value: blue ? (blue.supported ? (blue.powered ? `${blue.devices.filter((d) => d.gamepad && d.connected).length} controle(s) conectado(s)` : 'desligado') : 'pelo sistema') : '…', ...panel(PANELS.bluetooth) },
      { id: 'updates', label: 'Atualizações', icon: 'download', value: app.update?.available ? `nova: ${versionArrow(app.update)}` : 'em dia', ...panel(PANELS.updates) },
      { id: 'profiles', label: 'Perfis da casa', icon: 'lock', value: `${app.profile?.profiles.length ?? 1} perfil${(app.profile?.profiles.length ?? 1) === 1 ? '' : 's'} · modo infantil`, ...to({ name: 'profiles' }) },
      { id: 'saver', label: 'Protetor de tela', icon: 'resume', value: saver?.enabled ? `${saver.mode === 'snake' ? 'cobrinha' : 'vitrine'} · ${saver.minutes} min` : 'desligado', ...panel(PANELS.a11y) },
      { id: 'about', label: 'Sobre', icon: 'info', value: `versão ${app.version}`, ...panel(PANELS.about) },
    ]
    if (app.mode === 'desktop') items.push({ id: 'power', label: 'Energia', icon: 'power', value: 'minimizar, tela cheia, fechar', action: () => (app.power = true) })
    return items
  })

  const overview: Item = { id: 'overview', label: 'Resumo', icon: 'chart', value: 'suas horas e a semana' }
  const all = $derived([overview, ...mine, ...house])
  // item escolhido: o pedido por outra tela (atalho), o último visto ou o resumo
  let active = $state(untrack(() => (item && all.some((i) => i.id === item) ? item : (app.youItem ?? 'overview'))))
  const sel = $derived(all.find((i) => i.id === active) ?? overview)
  $effect(() => {
    app.youItem = active
  })

  // ao abrir: foco no item (ou já dentro do painel, se veio de um atalho)
  $effect(() => {
    untrack(() => tick().then(() => (item && sel.panel ? enter() : focusEl(listItem(active)))))
  })

  function listItem(id: string) {
    return document.querySelector<HTMLElement>(`.menu [data-item="${id}"]`)
  }

  function select(it: Item) {
    active = it.id
  }

  // Enter/clique: painel = entra nele; tela = abre; ação = faz
  function open(it: Item) {
    active = it.id
    if (it.screen) go(it.screen)
    else if (it.action) it.action()
    else tick().then(enter)
  }

  function enter() {
    const box = document.querySelector<HTMLElement>('.detail')
    const first = box?.querySelector<HTMLElement>('[data-nav-default]') ?? box?.querySelector<HTMLElement>('[data-nav]') ?? box
    if (first) focusEl(first)
  }

  function cycle(delta: number) {
    const i = all.findIndex((x) => x.id === active)
    const next = all[(i + delta + all.length) % all.length]
    play('move')
    focusEl(listItem(next.id))
  }

  $effect(() =>
    pushHandler((a) => {
      const el = current()
      const inside = !!el?.closest('.detail')
      // painel rolável em foco: cima/baixo rolam; na borda, voltam a navegar
      if (el?.hasAttribute('data-panel') && (a === 'up' || a === 'down')) {
        const before = el.scrollTop
        el.scrollBy({ top: a === 'down' ? 120 : -120 })
        if (el.scrollTop !== before) return true
      }
      if (a === 'back' && inside && !app.confirm && !app.prompt) {
        play('back')
        focusEl(listItem(active))
        return true
      }
      if (a === 'prevTab' || a === 'nextTab') {
        cycle(a === 'prevTab' ? -1 : 1)
        return true
      }
      return a === 'details' || a === 'menu'
    }),
  )
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="you">
    <nav class="menu" aria-label="Você e configurações">
      <header class="who">
        {#if me}<Avatar avatar={me.avatar} color={me.color} size="3.2rem" />{/if}
        <div>
          <p class="name pixel">{me?.name ?? 'Você'}</p>
          {#if kids}<p class="tag">Perfil infantil</p>{/if}
        </div>
      </header>
      {#each [{ title: '', items: [overview] }, { title: 'Só pra você', items: mine }, { title: 'Deste console', items: house }] as group (group.title)}
        {#if group.items.length}
          {#if group.title}<p class="group pixel">{group.title}</p>{/if}
          <ul>
            {#each group.items as it (it.id)}
              <li>
                <button
                  class="item"
                  data-nav
                  data-item={it.id}
                  data-nav-default={it.id === active || undefined}
                  aria-current={it.id === active ? 'page' : undefined}
                  onfocus={() => select(it)}
                  onclick={() => open(it)}
                >
                  <Icon name={it.icon} size="1.3em" />
                  <span class="text"><span class="label">{it.label}</span><span class="value muted">{it.value}</span></span>
                  {#if it.screen}<span class="go" aria-hidden="true">›</span>{/if}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      {/each}
      {#if kids}<p class="muted small">O resto do console quem muda é um adulto.</p>{/if}
    </nav>

    <!-- focável: dá pra rolar o painel com o controle mesmo sem botão dentro -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <section class="detail" aria-labelledby="you-title" tabindex={sel.panel ? 0 : undefined} data-panel={sel.panel ? '' : undefined} data-nav={sel.id === 'about' ? '' : undefined}>
      <p class="crumb muted" aria-hidden="true">Você ›</p>
      <h1 id="you-title" class="pixel">{sel.id === 'overview' ? (me?.name ?? 'Você') : sel.label}</h1>
      {#if sel.panel}
        {#key sel.id}<sel.panel />{/key}
      {:else if sel.id === 'overview'}
        <div class="card" style:--profile={me?.color ?? 'var(--accent)'}>
          {#if me}<Avatar avatar={me.avatar} color={me.color} size="7rem" />{/if}
      {#if kidsTime?.limited}
        <p class="today">{kidsTime.reason || `Hoje: restam ${playTime(kidsTime.minutesLeft * 60)}`}</p>
      {/if}
      <dl class="facts">
        <div><dt>horas</dt><dd class="hud">{hours}</dd></div>
        <div><dt>jogados</dt><dd class="hud">{played}</dd></div>
        <div><dt>favoritos</dt><dd class="hud">{favs}</dd></div>
      </dl>
      {#if summary && (summary.weekSeconds || summary.topGame)}
        {@const max = Math.max(1, ...summary.week.map((d) => d.seconds))}
        <section class="week" aria-labelledby="week-title">
          <h2 id="week-title">Esta semana · {summary.weekSeconds ? playTime(summary.weekSeconds) : 'nada ainda'}</h2>
          <ol class="bars" aria-label="Tempo jogado nos últimos 7 dias">
            {#each summary.week as d (d.date)}
              <li>
                <span class="bar" style:height={`${Math.max(4, (d.seconds / max) * 100)}%`} class:zero={!d.seconds}></span>
                <span class="day" aria-hidden="true">{d.label}</span>
                <span class="sr-only">{d.date}: {d.seconds ? playTime(d.seconds) : 'nada'}</span>
              </li>
            {/each}
          </ol>
          <dl class="more">
            {#if summary.topGame}<div><dt>Mais jogado</dt><dd>{summary.topGame}</dd></div>{/if}
            {#if summary.topSystem}<div><dt>Console preferido</dt><dd>{system(summary.topSystem)?.name ?? summary.topSystem}</dd></div>{/if}
            {#if summary.streak > 1}<div><dt>Sequência</dt><dd>{summary.streak} dias seguidos</dd></div>{/if}
            {#if cheevos?.connected}<div><dt>Conquistas</dt><dd>RetroAchievements: {cheevos.user}</dd></div>{/if}
          </dl>
        </section>
      {/if}
          <button class="btn primary" data-nav onclick={() => go({ name: 'profiles' })}><Icon name="resume" /> Trocar perfil</button>
        </div>
      {:else}
        <div class="preview">
          <p class="big-value">{sel.value}</p>
          <button class="btn primary" data-nav onclick={() => open(sel)}><Icon name={sel.icon} /> {sel.action ? sel.label : `Abrir ${sel.label.toLowerCase()}`}</button>
        </div>
      {/if}
    </section>
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'l1', label: 'Item anterior' },
      { button: 'r1', label: 'Próximo item' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .you {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(17rem, 22rem) 1fr;
    gap: 2.4rem;
    padding: 0.5rem 2.5rem 1rem;
  }
  .menu {
    overflow-y: auto;
    min-height: 0;
    padding: 0.3rem 0.6rem 1rem 0.3rem;
    scrollbar-width: thin;
  }
  .who {
    display: flex;
    gap: 0.8rem;
    align-items: center;
    margin-bottom: 0.6rem;
  }
  .who p {
    margin: 0;
  }
  .who .name {
    font-size: 1.1rem;
  }
  .group {
    font-size: 0.75rem;
    color: var(--accent-2);
    text-transform: uppercase;
    margin: 1.1rem 0 0.4rem;
  }
  .menu ul {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.35rem;
  }
  .item[aria-current='page'] {
    border-color: var(--accent);
    background: color-mix(in srgb, var(--accent) 16%, var(--surface));
  }
  .detail {
    overflow-y: auto;
    min-height: 0;
    padding: 0.3rem 0.4rem 1.5rem;
    outline-offset: -4px;
  }
  /* o título do item já está no topo: o do painel sairia repetido */
  .detail :global(.panel-body > h2:first-child),
  .detail :global(.panel-body > .head > h2) {
    display: none;
  }
  .crumb {
    margin: 0;
    font-size: 0.9rem;
  }
  .detail > h1 {
    font-size: 1.8rem;
    margin: 0.2rem 0 1.2rem;
  }
  .preview {
    display: grid;
    gap: 1.2rem;
    justify-items: start;
  }
  .big-value {
    font-size: 1.3rem;
    margin: 0;
  }
  .card {
    display: grid;
    justify-items: center;
    gap: 1rem;
    padding: 1.8rem 1.4rem;
    text-align: center;
    background: color-mix(in srgb, var(--profile) 14%, var(--surface));
    border: 2px solid var(--profile);
    box-shadow: 0 0 14px color-mix(in srgb, var(--profile) 55%, transparent), inset 0 0 40px color-mix(in srgb, var(--profile) 18%, transparent);
  }
  h1 {
    font-size: 1.7rem;
    overflow-wrap: anywhere;
  }
  .tag {
    color: var(--focus);
    border: 2px solid currentColor;
    padding: 0.1rem 0.6rem;
    font-size: 0.85rem;
  }
  .today {
    font-weight: 700;
  }
  .facts {
    margin: 0.3rem 0 0.6rem;
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0.6rem;
    width: 100%;
  }
  .facts div {
    display: flex;
    flex-direction: column-reverse;
    gap: 0.1rem;
  }
  .facts dt {
    font-size: 0.8rem;
    color: var(--muted);
  }
  .facts dd {
    margin: 0;
    font-size: 2rem;
    color: #fff;
    line-height: 1;
  }
  .week {
    width: 100%;
    display: grid;
    gap: 0.5rem;
    text-align: left;
  }
  .week h2 {
    font-size: 0.85rem;
    color: var(--muted);
    font-family: var(--font-body);
  }
  .bars {
    list-style: none;
    margin: 0;
    padding: 0;
    height: 4rem;
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 0.3rem;
    align-items: end;
  }
  .bars li {
    height: 100%;
    display: grid;
    grid-template-rows: 1fr auto;
    align-items: end;
    justify-items: center;
  }
  .bar {
    width: 100%;
    background: var(--profile);
    box-shadow: 0 0 6px var(--profile);
  }
  .bar.zero {
    opacity: 0.25;
    box-shadow: none;
  }
  .day {
    font-size: 0.7rem;
    color: var(--muted);
  }
  .more {
    margin: 0;
    display: grid;
    gap: 0.35rem;
  }
  .more div {
    display: flex;
    justify-content: space-between;
    gap: 0.6rem;
    font-size: 0.85rem;
  }
  .more dt {
    color: var(--muted);
  }
  .more dd {
    margin: 0;
    text-align: right;
    overflow-wrap: anywhere;
  }
  .item {
    width: 100%;
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: 1rem;
    padding: 0.9rem 1rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--surface) 85%, transparent);
    border: 2px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
    cursor: pointer;
  }
  .item:hover {
    border-color: var(--accent-2);
  }
  .item :global(svg) {
    color: var(--accent);
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    min-width: 0;
  }
  .label {
    font-weight: 700;
    font-size: 1.05rem;
  }
  .value {
    font-size: 0.9rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .go {
    font-family: var(--font-title);
    color: var(--accent-2);
  }
  .small {
    font-size: 0.9rem;
    margin-top: 0.8rem;
  }
</style>
