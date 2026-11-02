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
  import Activity from './Activity.svelte'
  import ImportScreen from './Import.svelte'
  import ProfileEditor from './ProfileEditor.svelte'
  import StatsScreen from './Stats.svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { activity, unread } from '../lib/activity.svelte'
  import { api } from '../lib/api'
  import { summarize, type ProfileSummary } from '../lib/profilestats'
  import { bytes, playTime } from '../lib/format'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app, currentProfile, go, isKids, system, type Screen } from '../lib/state.svelte'
  import type { CheevosInfo, EmulatorsInfo, KidsTimeInfo, RemoteInfo, StorageInfo } from '../lib/types'
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
  let cheevos = $state<CheevosInfo | null>(null)

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    api.stats('all').then((s) => (summary = summarize(s))).catch(() => {})
    if (isKids()) return
    api.remoteStatus().then((r) => (remote = r)).catch(() => {})
    api.storage().then((s) => (storage = s)).catch(() => {})
    api.emulatorStatus().then((e) => (emus = e)).catch(() => {})
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


  // Poucos itens, cada um com abas: nada escondido lá embaixo da lista.
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  type Sub = { id: string; label: string; panel?: Component<any>; props?: Record<string, unknown>; screen?: Screen; note?: string }
  type Group = { id: string; label: string; icon: IconName; value: string; subs: Sub[] }

  const groups = $derived.by((): Group[] => {
    const s = app.settings
    const a = s?.a11y
    const news = unread()
    const busy = activity.tasks.filter((t) => t.state === 'running').length
    const out: Group[] = [
      {
        id: 'me',
        label: 'Perfil',
        icon: 'chart',
        value: `${me?.name ?? ''} · ${hours} h jogadas`,
        subs: [
          { id: 'overview', label: 'Resumo' },
          { id: 'stats', label: 'Estatísticas', panel: StatsScreen },
          ...(kids
            ? []
            : [
                { id: 'profile', label: 'Editar perfil', panel: ProfileEditor, props: { id: me?.id } },
                { id: 'cheevos', label: 'Conquistas', panel: CheevosPanel },
              ]),
        ],
      },
      {
        id: 'activity',
        label: 'Central',
        icon: 'bell',
        value: busy ? `${busy} em andamento` : news ? `${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : 'downloads e avisos',
        subs: [{ id: 'activity', label: 'Central', panel: Activity }],
      },
    ]
    if (kids) return out
    const access = a
      ? [`texto ${Math.round(a.fontScale * 100)}%`, a.highContrast && 'alto contraste', a.reduceMotion && 'sem animações', !a.sounds && 'sem sons']
          .filter(Boolean)
          .join(' · ')
      : ''
    out.push(
      {
        id: 'controls',
        label: 'Controles',
        icon: 'pad',
        value: app.pads.length ? app.pads.map((p) => p.name).join(', ') : 'nenhum conectado',
        subs: [
          { id: 'controls', label: 'Botões', panel: ControlsPanel },
          { id: 'bluetooth', label: 'Bluetooth', panel: BluetoothPanel },
        ],
      },
      { id: 'a11y', label: 'Tela e som', icon: 'sliders', value: access, subs: [{ id: 'a11y', label: 'Tela e som', panel: A11yPanel }] },
      {
        id: 'games',
        label: 'Jogos',
        icon: 'folder',
        value: `${app.games.length} jogos · ${remote ? (remote.connected ? 'servidor conectado' : 'sem servidor') : '…'} · ${emuSummary()}`,
        subs: [
          { id: 'library', label: 'Biblioteca', panel: LibraryPanel },
          { id: 'import', label: 'Importar jogos', panel: ImportScreen },
          { id: 'server', label: 'Servidor RomM', panel: ServerPanel },
          { id: 'emulators', label: 'Emuladores e BIOS', panel: EmulatorsPanel },
        ],
      },
      {
        id: 'system',
        label: 'Sistema',
        icon: 'gear',
        value: [storage && storage.total > 0 ? `${bytes(storage.free)} livres` : '', app.update?.available ? `nova: ${versionArrow(app.update)}` : `versão ${app.version}`]
          .filter(Boolean)
          .join(' · '),
        subs: [
          { id: 'storage', label: 'Armazenamento', panel: StoragePanel },
          { id: 'updates', label: 'Atualizações', panel: UpdatesPanel },
          { id: 'profiles', label: 'Perfis da casa', screen: { name: 'profiles' }, note: 'Criar perfis, pôr PIN e ligar o modo infantil.' },
          { id: 'about', label: 'Sobre', panel: AboutPanel },
        ],
      },
    )
    return out
  })

  // ids antigos (atalhos de outras telas e da busca) -> grupo e aba
  const ALIAS: Record<string, string> = { saver: 'a11y', you: 'overview' }
  function locate(id: string): [string, string] | null {
    const want = ALIAS[id] ?? id
    for (const g of groups) {
      if (g.id === want) return [g.id, g.subs[0].id]
      const sub = g.subs.find((x) => x.id === want)
      if (sub) return [g.id, sub.id]
    }
    return null
  }

  const start = untrack(() => locate(item) ?? locate(app.youItem ?? '') ?? ['me', 'overview'])
  let active = $state(start[0])
  let subId = $state(start[1])
  const group = $derived(groups.find((g) => g.id === active) ?? groups[0])
  const sub = $derived(group.subs.find((x) => x.id === subId) ?? group.subs[0])
  $effect(() => {
    app.youItem = sub.id
  })

  // ao abrir: foco no item (ou já dentro do conteúdo, se veio de um atalho)
  $effect(() => {
    untrack(() => tick().then(() => (item ? enter() : focusEl(listItem(active)))))
  })

  function listItem(id: string) {
    return document.querySelector<HTMLElement>(`.menu [data-item="${id}"]`)
  }

  function select(g: Group) {
    if (active === g.id) return
    active = g.id
    subId = g.subs[0].id
  }

  // Enter/clique no item: entra no conteúdo (nas abas, se tiver)
  function open(g: Group) {
    select(g)
    tick().then(enter)
  }

  function enter() {
    const box = document.querySelector<HTMLElement>('.detail')
    const first =
      box?.querySelector<HTMLElement>('.subtabs [aria-current="page"]') ??
      box?.querySelector<HTMLElement>('.content [data-nav-default]') ??
      box?.querySelector<HTMLElement>('.content [data-nav]') ??
      box
    if (first) focusEl(first)
  }

  function pickSub(x: Sub) {
    subId = x.id
  }

  // Enter/clique na aba: entra no conteúdo dela
  function enterSub(x: Sub) {
    subId = x.id
    tick().then(() => {
      const box = document.querySelector<HTMLElement>('.detail .content')
      const first = box?.querySelector<HTMLElement>('[data-nav-default]') ?? box?.querySelector<HTMLElement>('[data-nav]')
      if (first) focusEl(first)
    })
  }

  // L1/R1: troca de aba no grupo; sem abas, troca de item
  function cycle(delta: number) {
    play('move')
    if (group.subs.length > 1) {
      const i = group.subs.findIndex((x) => x.id === sub.id)
      subId = group.subs[(i + delta + group.subs.length) % group.subs.length].id
      tick().then(() => focusEl(document.querySelector<HTMLElement>('.subtabs [aria-current="page"]')))
      return
    }
    const i = groups.findIndex((g) => g.id === active)
    focusEl(listItem(groups[(i + delta + groups.length) % groups.length].id))
  }

  // .pre: registra antes dos painéis, que assim ouvem primeiro (a prévia de
  // controles usa L1/R1 pra testar, sem trocar de aba)
  $effect.pre(() =>
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
          <p class="name sub">{me?.name ?? 'Você'}</p>
          {#if kids}<p class="tag">Perfil infantil</p>{/if}
        </div>
      </header>
      <ul>
        {#each groups as g (g.id)}
          <li>
            <button
              class="item"
              data-nav
              data-item={g.id}
              data-nav-default={g.id === active || undefined}
              aria-current={g.id === active ? 'page' : undefined}
              onfocus={() => select(g)}
              onclick={() => open(g)}
            >
              <Icon name={g.icon} size="1.3em" />
              <span class="text"><span class="label">{g.label}</span><span class="value muted">{g.value}</span></span>
            </button>
          </li>
        {/each}
      </ul>
      {#if kids}<p class="muted small">O resto do console quem muda é um adulto.</p>{/if}
    </nav>

    <!-- focável: dá pra rolar o painel com o controle mesmo sem botão dentro -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <section class="detail" aria-labelledby="you-title" tabindex={sub.panel ? 0 : undefined} data-panel={sub.panel ? '' : undefined} data-nav={sub.id === 'about' ? '' : undefined}>
      <p class="crumb muted" aria-hidden="true">Você › {group.label}</p>
      <h1 id="you-title" class="pixel">{sub.id === 'overview' ? (me?.name ?? 'Você') : group.subs.length > 1 ? sub.label : group.label}</h1>
      {#if group.subs.length > 1}
        <div class="subtabs" role="group" aria-label={`Seções de ${group.label}`}>
          {#each group.subs as x (x.id)}
            <button class="btn small" data-nav aria-current={x.id === sub.id ? 'page' : undefined} onfocus={() => pickSub(x)} onclick={() => enterSub(x)}>{x.label}</button>
          {/each}
        </div>
      {/if}
      <div class="content">
      {#if sub.panel}
        <!-- telas inteiras (central, perfil, estatísticas, importar) entram embutidas -->
        {#key sub.id}<sub.panel {...sub.props ?? {}} embedded onclose={() => focusEl(listItem(active))} />{/key}
      {:else if sub.id === 'overview'}
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
      {:else if sub.screen}
        <div class="preview">
          <p class="big-value">{sub.note}</p>
          <button class="btn primary" data-nav onclick={() => sub.screen && go(sub.screen)}><Icon name="lock" /> Abrir {sub.label.toLowerCase()}</button>
        </div>
      {/if}
      </div>
    </section>
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Abrir' },
      { button: 'l1', label: 'Aba anterior' },
      { button: 'r1', label: 'Próxima aba' },
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
  .subtabs {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
    margin: -0.4rem 0 1.4rem;
  }
  .subtabs [aria-current='page'] {
    background: var(--accent-2);
    color: #050008;
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
  .small {
    font-size: 0.9rem;
    margin-top: 0.8rem;
  }
</style>
