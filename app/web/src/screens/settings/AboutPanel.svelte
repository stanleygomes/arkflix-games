<script lang="ts">
  import Glyph from '../../components/Glyph.svelte'
  import Icon from '../../components/Icon.svelte'
  import { api, on } from '../../lib/api'
  import { bytes } from '../../lib/format'
  import { app, glyphFamily, toast } from '../../lib/state.svelte'
  import type { ScriptsInfo, UpdateInfo } from '../../lib/types'
  import { saveSoon } from './save.svelte'

  let update = $state<UpdateInfo | null>(null)
  let checking = $state(false)
  $effect(() => {
    api.updateStatus().then((u) => (update = u)).catch(() => {})
    return on<UpdateInfo>('update', (u) => (update = u))
  })

  async function check() {
    checking = true
    try {
      update = await api.checkUpdate()
      if (!update.available) toast(`Você está na versão mais nova (${update.current})`, 'ok', 3000)
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      checking = false
    }
  }

  async function download() {
    try {
      await api.downloadUpdate()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function toggleAuto() {
    if (app.settings) saveSoon({ ...app.settings, autoUpdate: !app.settings.autoUpdate })
  }

  const family = $derived(glyphFamily())
  let scripts = $state<ScriptsInfo | null>(null)
  $effect(() => {
    api.scripts().then((s) => (scripts = s)).catch(() => {})
  })
  const labels: Record<string, string> = { 'game-start': 'Ao abrir um jogo', 'game-end': 'Ao fechar um jogo' }
</script>

<div class="panel-body">
  <h2>Sobre</h2>
  <dl class="facts">
    <dt>Versão</dt>
    <dd class="pixel">{app.version}</dd>
    <dt>Modo</dt>
    <dd>{app.mode === 'desktop' ? 'App desktop' : 'Navegador'}</dd>
    <dt>Jogos</dt>
    <dd>{app.games.length} em {app.systems.filter((s) => app.games.some((g) => g.system === s.slug)).length} consoles</dd>
  </dl>

  <section class="update" aria-labelledby="update-title">
    <h3 id="update-title">Atualização</h3>
    {#if update?.state === 'ready'}
      <p class="status"><Icon name="check" /> A versão <strong>{update.latest}</strong> já baixou: instala quando você fechar o app.</p>
    {:else if update?.state === 'downloading'}
      <p>Baixando a versão {update.latest}… {Math.round(update.progress * 100)}%</p>
      <div class="energy" role="progressbar" aria-label="Baixando atualização" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(update.progress * 100)}>
        <span style:width={`${Math.floor(update.progress * 20) * 5}%`}></span>
      </div>
    {:else if update?.available}
      <p class="status"><Icon name="download" /> Versão nova: <strong>{update.latest}</strong>{update.size ? ` (${bytes(update.size)})` : ''}.</p>
      {#if update.notes}<pre class="notes">{update.notes}</pre>{/if}
      {#if update.error}<p class="error" role="alert">{update.error}</p>{/if}
      <button class="btn primary" data-nav onclick={download}><Icon name="download" /> Baixar agora</button>
    {:else}
      <p class="muted">{update?.error || 'Nenhuma versão nova encontrada ainda.'}</p>
    {/if}
    <div class="actions">
      <button class="btn" data-nav disabled={checking || update?.state === 'downloading'} onclick={check}><Icon name="resume" /> {checking ? 'Procurando…' : 'Procurar atualização'}</button>
      {#if app.settings}
        <button class="btn" data-nav role="switch" aria-checked={app.settings.autoUpdate} onclick={toggleAuto}>Atualizar sozinho</button>
      {/if}
    </div>
  </section>

  <h3>Na interface</h3>
  <ul class="keys">
    <li><Glyph button="south" {family} /> abrir / confirmar</li>
    <li><Glyph button="east" {family} /> voltar</li>
    <li><Glyph button="north" {family} /> favorito (ou apagar o ponto salvo em foco)</li>
    <li><Glyph button="west" {family} /> detalhes do jogo</li>
    <li><Glyph button="l1" {family} /> <Glyph button="r1" {family} /> trocar console / seção</li>
    <li><Glyph button="start" {family} /> você e configurações</li>
    <li><Glyph button="home" {family} /> guia rápido (até com o jogo aberto) · G</li>
    <li>Teclado: setas, Enter, Esc, F (favorito), I (detalhes), Q/E (seções), F11 (tela cheia)</li>
  </ul>

  <h3>Dentro do jogo (todos os consoles)</h3>
  <ul class="keys">
    <li><Glyph button="select" {family} /> + <Glyph button="north" {family} /> menu do emulador (pausa, salvar, carregar) · Esc</li>
    <li><Glyph button="select" {family} /> + <Glyph button="start" {family} /> sair e voltar pro app · F10</li>
  </ul>
  <h3>Atalhos rápidos (menos PSP e PS2)</h3>
  <ul class="keys">
    <li><Glyph button="select" {family} /> + <Glyph button="r1" {family} /> salvar · F2</li>
    <li><Glyph button="select" {family} /> + <Glyph button="l1" {family} /> carregar · F4</li>
  </ul>

  <h3>Scripts de evento</h3>
  {#if scripts}
    <p class="muted">
      Executáveis em <code>{scripts.dir}/&lt;evento&gt;/</code> rodam em segundo plano, em ordem alfabética, recebendo o caminho da ROM, o
      nome e o console (ao fechar, também os segundos jogados). Log em <code>{scripts.log}</code>.
    </p>
    <dl class="facts" aria-label="Scripts instalados">
      {#each Object.entries(scripts.events) as [event, names] (event)}
        <dt>{labels[event] ?? event}</dt>
        <dd>{names.length ? names.join(', ') : 'nenhum'}</dd>
      {/each}
    </dl>
  {/if}

  <h3>Créditos</h3>
  <p class="muted">
    Fontes Pixelify Sans e Atkinson Hyperlegible (SIL Open Font License). Emuladores: RetroArch, PPSSPP e PCSX2, cada um com a
    sua licença (veja o README). Sons e música sintetizados pelo próprio app.
  </p>
</div>

<style>
  .update {
    display: grid;
    gap: 0.7rem;
    justify-items: start;
  }
  .update p {
    margin: 0;
  }
  .status {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .notes {
    max-height: 8rem;
    overflow-y: auto;
    white-space: pre-wrap;
    font: inherit;
    font-size: 0.85rem;
    color: var(--muted);
    margin: 0;
  }
  .update .energy {
    width: 100%;
  }
  .actions {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  h3 {
    font-size: 1.1rem;
    margin-top: 0.5rem;
  }
  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.4rem 1.2rem;
    margin: 0;
  }
  dt {
    color: var(--muted);
  }
  dd {
    margin: 0;
  }
  code {
    color: var(--accent-2);
    overflow-wrap: anywhere;
  }
  .keys {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  .keys li {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    flex-wrap: wrap;
  }
</style>
