<script lang="ts">
  // Central: downloads e tarefas (com cancelar) e o histórico dos avisos.
  import { tick } from 'svelte'
  import DownloadOptions from '../components/DownloadOptions.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { active, activity, cancelTask, cancellable, clearFinished, downloadNow, markRead, pauseDownload, resumeDownload } from '../lib/activity.svelte'
  import { downloadGame } from '../lib/downloads.svelte'
  import { ago } from '../lib/format'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app, game, isKids } from '../lib/state.svelte'
  import type { Task } from '../lib/activity.svelte'

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    if (!embedded) tick().then(focusDefault)
    // abriu a central: os avisos foram vistos (ao sair, pra a bolinha não sumir antes)
    return markRead
  })

  const now = $derived(Date.now())
  const when = (ms: number) => ago(Math.floor(ms / 1000), now / 1000)
  const status: Record<Task['state'], string> = {
    running: 'em andamento',
    queued: 'na fila',
    waiting: 'esperando a madrugada',
    paused: 'pausado',
    done: 'pronto',
    error: 'falhou',
    cancelled: 'cancelado',
  }
  let options = $state(false)
  let optionsButton = $state<HTMLButtonElement>()

  function closeOptions() {
    options = false
    tick().then(() => optionsButton?.focus())
  }

  // o primeiro botão de cada download: pausar, continuar ou baixar agora
  function main(t: Task) {
    if (!t.id.startsWith('download:') || isKids()) return undefined
    if (t.state === 'running') return { icon: 'pause' as const, label: 'Pausar', run: () => pauseDownload(t.id) }
    if (t.state === 'paused') return { icon: 'play' as const, label: 'Continuar', run: () => resumeDownload(t.id) }
    if (t.state === 'waiting') return { icon: 'download' as const, label: 'Baixar agora mesmo assim', run: () => downloadNow(t.id) }
    return undefined
  }

  // download que falhou ou foi cancelado: dá pra tentar de novo
  function retry(t: Task) {
    const g = t.id.startsWith('download:') ? game(t.id.slice(9)) : undefined
    return (t.state === 'error' || t.state === 'cancelled') && g?.remote ? g : undefined
  }
  // embedded: aberta como painel da tela Você (lista sempre visível)
  let { embedded = false }: { embedded?: boolean } = $props()
</script>

<div class="page" class:embedded>
  {#if !embedded}<TopBar actions={false} />{/if}
  <main class="central" aria-labelledby={embedded ? undefined : 'central-title'}>
    {#if !embedded}<h1 id="central-title" class="pixel">Central</h1>{/if}
    <section class="col" aria-labelledby="tasks-title">
      <header>
        <h2 id="tasks-title" class="pixel">Downloads e tarefas</h2>
        <div class="tools">
          {#if activity.tasks.some((t) => !active(t))}
            <button class="btn small" data-nav onclick={clearFinished}><Icon name="trash" /> Limpar concluídas</button>
          {/if}
          {#if !isKids()}
            <button class="btn small" data-nav bind:this={optionsButton} onclick={() => (options = true)}><Icon name="sliders" /> Opções de download</button>
          {/if}
        </div>
      </header>
      {#if activity.tasks.length === 0}
        <p class="empty muted">Nada baixando agora. Jogos do servidor, capas, saves e emuladores aparecem aqui.</p>
      {:else}
        <ul>
          {#each activity.tasks as t (t.id)}
            <li class="task {t.state}">
              <div class="line">
                <span class="label">{t.label}</span>
                <span class="state">{status[t.state]} · {when(t.at)}</span>
              </div>
              {#if t.state === 'running' || (t.state === 'paused' && t.progress > 0)}
                <div class="energy" role="progressbar" aria-label={t.label} aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(t.progress * 100)}>
                  <span style:width={`${Math.floor(t.progress * 20) * 5}%`}></span>
                </div>
              {:else if t.message && !active(t)}
                <p class="msg muted">{t.message}</p>
              {/if}
              {#if cancellable(t)}
                {@const first = main(t)}
                <div class="buttons">
                  {#if first}
                    <button class="btn small" data-nav data-nav-default onclick={first.run}><Icon name={first.icon} /> {first.label}</button>
                  {/if}
                  <button class="btn small" data-nav data-nav-default={first ? undefined : ''} onclick={() => cancelTask(t.id)}><Icon name="close" /> Cancelar</button>
                </div>
              {:else if retry(t)}
                <button class="btn small" data-nav onclick={() => downloadGame(retry(t))}><Icon name="download" /> Baixar de novo</button>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </section>

    <section class="col" aria-labelledby="notices-title">
      <header>
        <h2 id="notices-title" class="pixel">Avisos</h2>
        {#if app.notices.length}
          <button class="btn small" data-nav data-nav-default={activity.tasks.length ? undefined : ''} onclick={() => (app.notices = [])}><Icon name="trash" /> Limpar avisos</button>
        {/if}
      </header>
      {#if app.notices.length === 0}
        <p class="empty muted">Nenhum aviso por enquanto.</p>
      {:else}
        <ul>
          {#each app.notices as n (n.id)}
            <li class="notice {n.kind}" class:new={!n.read}>
              <Icon name={n.kind === 'error' ? 'warn' : n.kind === 'ok' ? 'check' : 'info'} />
              <span class="text">{n.text}{#if n.count > 1}<span class="count"> · {n.count} avisos</span>{/if}</span>
              <span class="state">{when(n.at)}</span>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  </main>
  {#if options}
    <DownloadOptions onclose={closeOptions} />
  {/if}
  {#if !embedded}<HelpBar hints={[{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />{/if}
</div>

<style>
  /* embutida na tela Você: sem barra, sem rodapé, sem rolagem própria */
  .page.embedded {
    height: auto;
    display: block;
  }
  .page.embedded > main {
    padding: 0;
    overflow: visible;
  }

  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .central {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: grid;
    grid-template-columns: 1fr 1fr;
    grid-template-rows: auto 1fr;
    gap: 1rem 2.4rem;
    padding: 0.5rem 2.5rem 1.5rem;
    align-items: start;
  }
  h1 {
    grid-column: 1 / -1;
    font-size: 2rem;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    margin-bottom: 0.8rem;
  }
  h2 {
    font-size: 0.95rem;
    color: var(--accent-2);
    text-shadow: 0 0 8px var(--accent-2);
    text-transform: uppercase;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0.3rem;
    display: grid;
    gap: 0.6rem;
  }
  li {
    padding: 0.8rem 1rem;
    background: color-mix(in srgb, var(--surface) 85%, transparent);
    border: 2px solid color-mix(in srgb, var(--accent-2) 40%, transparent);
  }
  .task {
    display: grid;
    gap: 0.5rem;
    justify-items: start;
  }
  .task.error,
  .notice.error {
    border-color: #ff6b6b;
  }
  .task.done,
  .notice.ok {
    border-color: color-mix(in srgb, #5dc44f 70%, transparent);
  }
  .line {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    width: 100%;
  }
  .label {
    font-weight: 700;
  }
  .tools,
  .buttons {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
  }
  .tools {
    justify-content: flex-end;
  }
  .task.paused {
    border-style: dashed;
  }
  .state {
    color: var(--muted);
    font-size: 0.85rem;
    white-space: nowrap;
  }
  .energy {
    width: 100%;
  }
  .msg {
    margin: 0;
    font-size: 0.9rem;
    white-space: pre-line;
  }
  .notice {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.7rem;
    align-items: start;
  }
  .notice.new {
    box-shadow: inset 4px 0 0 var(--focus);
  }
  .count {
    color: var(--muted);
  }
  .empty {
    padding: 0.3rem;
  }
</style>
