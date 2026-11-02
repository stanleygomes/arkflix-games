<script lang="ts">
  // Atualizações: versão atual → nova, novidades, "Atualizar e reiniciar"
  // num clique, procurar de novo e o "atualizar sozinho".
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { ago, bytes } from '../../lib/format'
  import { app, isKids, toast } from '../../lib/state.svelte'
  import { shortVersion, updateNow } from '../../lib/update'
  import { saveSoon } from './save.svelte'

  let checking = $state(false)
  const u = $derived(app.update)

  async function check() {
    checking = true
    try {
      app.update = await api.checkUpdate()
      if (!app.update.available) toast(`Você já está na versão mais nova (${shortVersion(app.update.current)})`, 'ok', 3000)
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      checking = false
    }
  }

  async function downloadOnly() {
    try {
      await api.downloadUpdate()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  function toggleAuto() {
    if (app.settings) saveSoon({ ...app.settings, autoUpdate: !app.settings.autoUpdate })
  }

  const busy = $derived(u?.state === 'downloading' || u?.state === 'restarting')
</script>

<div class="panel-body">
  <h2>Atualizações</h2>

  {#if u?.available}
    <section class="card new" aria-labelledby="upd-new">
      <p class="tag pixel">Versão nova</p>
      <h3 id="upd-new" class="versions">
        <span class="old">{shortVersion(u.current)}</span>
        <span class="arrow" aria-label="para">→</span>
        <span class="neu pixel">{u.latest}</span>
      </h3>
      {#if u.size}<p class="muted small">{bytes(u.size)} pra baixar</p>{/if}
      {#if u.state === 'downloading'}
        <p>{u.restart ? 'Baixando; reinicia sozinho no fim…' : 'Baixando…'} {Math.round(u.progress * 100)}%</p>
        <div class="energy" role="progressbar" aria-label="Baixando atualização" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(u.progress * 100)}>
          <span style:width={`${Math.floor(u.progress * 20) * 5}%`}></span>
        </div>
      {:else if u.state === 'ready'}
        <p class="ok"><Icon name="check" /> Já baixou. Reinicie agora ou ela instala quando você fechar o app.</p>
      {/if}
      {#if u.error}<p class="error" role="alert">{u.error}</p>{/if}
      {#if isKids()}
        <p class="muted">Peça pra um adulto atualizar.</p>
      {:else}
        <div class="actions">
          <button class="btn primary" data-nav data-nav-default disabled={busy || u.asset === ''} onclick={updateNow}>
            <Icon name="download" /> {u.state === 'ready' ? 'Reiniciar e atualizar' : 'Atualizar e reiniciar'}
          </button>
          {#if u.state === 'idle' || u.state === 'error'}
            <button class="btn" data-nav disabled={u.asset === ''} onclick={downloadOnly}>Só baixar (instala ao fechar)</button>
          {/if}
        </div>
        {#if u.asset === ''}<p class="muted small">Essa versão não tem instalador pra este sistema ainda.</p>{/if}
      {/if}
      {#if u.notes}
        <h4>Novidades</h4>
        <pre class="notes">{u.notes}</pre>
      {/if}
      {#if u.url}<p class="muted small">Detalhes: {u.url}</p>{/if}
    </section>
  {:else}
    <section class="card" aria-labelledby="upd-cur">
      <p class="tag pixel ok-tag">Em dia</p>
      <h3 id="upd-cur" class="versions"><span class="neu pixel">{shortVersion(u?.current ?? app.version)}</span></h3>
      <p class="muted">{u?.error || 'Você está na versão mais nova que encontramos.'}</p>
    </section>
  {/if}

  <div class="actions">
    <button class="btn" data-nav data-nav-default={u?.available ? undefined : ''} disabled={checking || busy} onclick={check}>
      <Icon name="resume" /> {checking ? 'Procurando…' : 'Procurar atualizações'}
    </button>
    {#if app.settings && !isKids()}
      <button class="btn" data-nav role="switch" aria-checked={app.settings.autoUpdate} onclick={toggleAuto}>Baixar sozinho</button>
    {/if}
  </div>
  <p class="muted small">
    {u?.checked ? `Última consulta ${ago(u.checked)}.` : 'Ainda não consultou nesta sessão.'}
    O app procura ao abrir e a cada 6 horas. Com "Baixar sozinho", a versão nova já vem baixada e instala ao fechar.
  </p>
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  .card {
    display: grid;
    gap: 0.7rem;
    justify-items: start;
    padding: 1.2rem 1.4rem;
    border: 2px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
  }
  .card.new {
    border-color: var(--focus);
    box-shadow: 0 0 14px color-mix(in srgb, var(--focus) 40%, transparent);
  }
  .card p {
    margin: 0;
  }
  .tag {
    font-size: 0.75rem;
    color: var(--focus);
  }
  .ok-tag {
    color: #5dc44f;
  }
  .versions {
    display: flex;
    align-items: center;
    gap: 1rem;
    font-size: 1.8rem;
    margin: 0;
  }
  .old {
    color: var(--muted);
    font-size: 1.3rem;
  }
  .arrow {
    color: var(--accent-2);
  }
  .neu {
    color: #fff;
    text-shadow: 0 0 10px var(--accent);
  }
  .card .energy {
    width: 100%;
  }
  .ok {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    color: #5dc44f;
  }
  h4 {
    margin: 0.4rem 0 0;
    font-size: 1rem;
  }
  .notes {
    max-height: 12rem;
    width: 100%;
    overflow-y: auto;
    white-space: pre-wrap;
    font: inherit;
    font-size: 0.9rem;
    color: var(--muted);
    margin: 0;
  }
  .actions {
    display: flex;
    gap: 0.6rem;
    flex-wrap: wrap;
  }
  .small {
    font-size: 0.85rem;
    overflow-wrap: anywhere;
  }
</style>
