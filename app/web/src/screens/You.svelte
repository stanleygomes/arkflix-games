<script lang="ts">
  // "Você": perfil e configurações num lugar só, separados pelo que vale só
  // pra pessoa (perfil, controle, acessibilidade, estatísticas) e pelo que vale
  // pro console todo (biblioteca, servidor, armazenamento, perfis da casa...).
  // Cada item mostra o valor atual e abre a tela dele.
  import { tick } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { activity, unread } from '../lib/activity.svelte'
  import { api } from '../lib/api'
  import { avatarName } from '../lib/avatars'
  import { bytes } from '../lib/format'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app, currentProfile, go, isKids, type Screen } from '../lib/state.svelte'
  import type { BTInfo, CheevosInfo, EmulatorsInfo, RemoteInfo, RemotePlayInfo, StorageInfo } from '../lib/types'
  import { versionArrow } from '../lib/update'

  const me = $derived(currentProfile())
  const kids = $derived(isKids())
  let remote = $state<RemoteInfo | null>(null)
  let storage = $state<StorageInfo | null>(null)
  let emus = $state<EmulatorsInfo | null>(null)
  let blue = $state<BTInfo | null>(null)
  let cheevos = $state<CheevosInfo | null>(null)
  let stream = $state<RemotePlayInfo | null>(null)

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    tick().then(focusDefault)
    if (isKids()) return
    api.remoteStatus().then((r) => (remote = r)).catch(() => {})
    api.storage().then((s) => (storage = s)).catch(() => {})
    api.emulatorStatus().then((e) => (emus = e)).catch(() => {})
    api.btStatus().then((b) => (blue = b)).catch(() => {})
    api.cheevosStatus().then((c) => (cheevos = c)).catch(() => {})
    api.remotePlayStatus().then((r) => (stream = r)).catch(() => {})
  })

  const hours = $derived(Math.floor(app.games.reduce((t, g) => t + (g.playTime || 0), 0) / 3600))
  const played = $derived(app.games.filter((g) => g.lastPlayed > 0).length)
  const favs = $derived(app.games.filter((g) => g.favorite).length)

  function emuSummary(): string {
    if (!emus) return '…'
    const miss = emus.emulators.filter((e) => !e.installed).length + emus.cores.filter((c) => !c.installed).length
    return miss ? `faltam ${miss} (instalar)` : 'tudo instalado'
  }

  type Item = { id: string; label: string; icon: IconName; value: string; open: () => void }
  const to = (s: Screen) => () => go(s)

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
      { id: 'activity', label: 'Central', icon: 'bell', value: busy ? `${busy} em andamento` : news ? `${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : 'downloads e avisos', open: to({ name: 'activity' }) },
      { id: 'stats', label: 'Estatísticas', icon: 'chart', value: `${hours} h jogadas · ${played} jogos`, open: to({ name: 'stats' }) },
    ]
    if (!kids) {
      items.splice(1, 0, { id: 'profile', label: 'Perfil', icon: 'pencil', value: `${me?.name ?? ''} · ${avatarName(me?.avatar ?? '')}${me?.hasPIN ? ' · com PIN' : ''}`, open: to({ name: 'profile', id: me?.id }) })
      items.push(
        { id: 'controls', label: 'Controles', icon: 'pad', value: pad, open: to({ name: 'settings', tab: 'controls' }) },
        { id: 'a11y', label: 'Acessibilidade e som', icon: 'sliders', value: access, open: to({ name: 'settings', tab: 'a11y' }) },
        { id: 'cheevos', label: 'Conquistas', icon: 'trophy', value: cheevos ? (cheevos.connected ? `RetroAchievements: ${cheevos.user}` : 'desligadas') : '…', open: to({ name: 'settings', tab: 'cheevos' }) },
      )
    }
    return items
  })

  const house = $derived.by((): Item[] => {
    if (kids) return []
    const s = app.settings
    const saver = s?.screensaver
    const items: Item[] = [
      { id: 'library', label: 'Biblioteca', icon: 'folder', value: `${s?.libraryDirs.length ?? 0} pasta${s?.libraryDirs.length === 1 ? '' : 's'} · ${app.games.length} jogos · capas ${s?.covers.enabled ? 'ligadas' : 'desligadas'}`, open: to({ name: 'settings', tab: 'library' }) },
      { id: 'import', label: 'Importar jogos', icon: 'plus', value: 'de qualquer pasta ou pendrive', open: to({ name: 'import' }) },
      { id: 'server', label: 'Servidor RomM', icon: 'cloud', value: remote ? (remote.connected ? `conectado · ${remote.games} jogos` : 'offline') : '…', open: to({ name: 'settings', tab: 'server' }) },
      { id: 'storage', label: 'Armazenamento', icon: 'disc', value: storage ? (storage.total > 0 ? `${bytes(storage.free)} livres de ${bytes(storage.total)}` : 'jogos: ' + bytes(storage.games)) : '…', open: to({ name: 'settings', tab: 'storage' }) },
      { id: 'emulators', label: 'Emuladores', icon: 'pad', value: emuSummary(), open: to({ name: 'settings', tab: 'emulators' }) },
      { id: 'bluetooth', label: 'Bluetooth', icon: 'bluetooth', value: blue ? (blue.supported ? (blue.powered ? `${blue.devices.filter((d) => d.gamepad && d.connected).length} controle(s) conectado(s)` : 'desligado') : 'pelo sistema') : '…', open: to({ name: 'settings', tab: 'bluetooth' }) },
      { id: 'remoteplay', label: 'Jogar em outro aparelho', icon: 'tv', value: stream ? (stream.running ? `ligado · ${stream.lanIP || 'nesta rede'}` : stream.installed ? 'desligado' : 'TV ou celular com Moonlight') : '…', open: to({ name: 'settings', tab: 'remoteplay' }) },
      { id: 'updates', label: 'Atualizações', icon: 'download', value: app.update?.available ? `nova: ${versionArrow(app.update)}` : 'em dia', open: to({ name: 'settings', tab: 'updates' }) },
      { id: 'profiles', label: 'Perfis da casa', icon: 'lock', value: `${app.profile?.profiles.length ?? 1} perfil${(app.profile?.profiles.length ?? 1) === 1 ? '' : 's'} · modo infantil`, open: to({ name: 'profiles' }) },
      { id: 'saver', label: 'Protetor de tela', icon: 'resume', value: saver?.enabled ? `${saver.mode === 'snake' ? 'cobrinha' : 'vitrine'} · ${saver.minutes} min` : 'desligado', open: to({ name: 'settings', tab: 'a11y' }) },
      { id: 'about', label: 'Sobre', icon: 'info', value: `versão ${app.version}`, open: to({ name: 'settings', tab: 'about' }) },
    ]
    if (app.mode === 'desktop') items.push({ id: 'power', label: 'Energia', icon: 'power', value: 'minimizar, tela cheia, fechar', open: () => (app.power = true) })
    return items
  })
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="you" aria-labelledby="you-title">
    <aside class="card" style:--profile={me?.color ?? 'var(--accent)'}>
      {#if me}<Avatar avatar={me.avatar} color={me.color} size="9rem" />{/if}
      <h1 id="you-title" class="pixel">{me?.name ?? 'Você'}</h1>
      {#if kids}<p class="tag">Perfil infantil</p>{/if}
      <dl class="facts">
        <div><dt>horas</dt><dd class="hud">{hours}</dd></div>
        <div><dt>jogados</dt><dd class="hud">{played}</dd></div>
        <div><dt>favoritos</dt><dd class="hud">{favs}</dd></div>
      </dl>
      <button class="btn primary" data-nav data-nav-default onclick={() => go({ name: 'profiles' })}><Icon name="resume" /> Trocar perfil</button>
    </aside>

    <section class="col" aria-labelledby="mine-title">
      <h2 id="mine-title" class="pixel">Só pra você</h2>
      <ul>
        {#each mine as it (it.id)}
          <li>
            <button class="item" data-nav data-item={it.id} onclick={it.open}>
              <Icon name={it.icon} size="1.4em" />
              <span class="text"><span class="label">{it.label}</span><span class="value muted">{it.value}</span></span>
              <span class="go" aria-hidden="true">›</span>
            </button>
          </li>
        {/each}
      </ul>
      {#if kids}<p class="muted small">O resto do console quem muda é um adulto.</p>{/if}
    </section>

    {#if house.length}
      <section class="col" aria-labelledby="house-title">
        <h2 id="house-title" class="pixel">Deste console</h2>
        <ul>
          {#each house as it (it.id)}
            <li>
              <button class="item" data-nav data-item={it.id} onclick={it.open}>
                <Icon name={it.icon} size="1.4em" />
                <span class="text"><span class="label">{it.label}</span><span class="value muted">{it.value}</span></span>
                <span class="go" aria-hidden="true">›</span>
              </button>
            </li>
          {/each}
        </ul>
      </section>
    {/if}
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Abrir' }, { button: 'east', label: 'Voltar' }]} />
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
    overflow-y: auto;
    display: grid;
    grid-template-columns: minmax(15rem, 20rem) 1fr 1fr;
    gap: 2.2rem;
    padding: 0.5rem 2.5rem 1.5rem;
    align-items: start;
  }
  .card {
    position: sticky;
    top: 0;
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
  .col h2 {
    font-size: 0.95rem;
    color: var(--accent-2);
    text-shadow: 0 0 8px var(--accent-2);
    text-transform: uppercase;
    margin: 0.4rem 0 1rem;
  }
  .col ul {
    list-style: none;
    margin: 0;
    padding: 0.3rem;
    display: grid;
    gap: 0.7rem;
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
  @media (max-width: 1100px) {
    .you {
      grid-template-columns: minmax(14rem, 18rem) 1fr;
    }
    .col:last-child {
      grid-column: 2;
    }
  }
</style>
