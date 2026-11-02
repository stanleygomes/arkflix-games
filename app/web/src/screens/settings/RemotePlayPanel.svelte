<script lang="ts">
  // Jogar em outro aparelho: o Sunshine transmite este console e o Moonlight
  // (TV, celular, outro PC) recebe. O app só instala, liga/desliga e mostra
  // o endereço; o PIN do Moonlight se digita na página local do Sunshine.
  import Icon from '../../components/Icon.svelte'
  import { activity, cancelTask } from '../../lib/activity.svelte'
  import { api, on } from '../../lib/api'
  import { play } from '../../lib/sound'
  import { app, toast } from '../../lib/state.svelte'
  import type { RemotePlayInfo } from '../../lib/types'

  let info = $state<RemotePlayInfo | null>(null)
  let error = $state('')

  $effect(() => {
    api.remotePlayStatus().then((i) => (info = i)).catch((err) => (error = (err as Error).message))
    return on<RemotePlayInfo>('remoteplay', (i) => (info = i))
  })

  const task = $derived(activity.tasks.find((t) => t.id === 'remoteplay' && t.state === 'running'))

  async function run(fn: () => Promise<unknown>, ok = '') {
    try {
      await fn()
      play('accept')
      if (ok) toast(ok, 'info', 2500, 'remoteplay')
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  // no navegador, os links abrem aqui mesmo; no desktop, no navegador do sistema
  function open(url: string, viaApp: () => Promise<void>, ok: string) {
    if (app.mode === 'web') {
      window.open(url, '_blank', 'noopener')
      return
    }
    run(viaApp, ok)
  }

  function install() {
    if (!info) return
    if (info.autoInstall) return run(api.installRemotePlay, 'Instalando o Sunshine…')
    open(info.download ?? '', api.installRemotePlay, 'Abri a página de download: instale e volte aqui')
  }
</script>

<div class="panel-body">
  <header class="head">
    <h2>Jogar em outro aparelho</h2>
    {#if info?.supported && !task}
      {#if !info.installed}
        <button class="btn primary" data-nav data-nav-default onclick={install}>
          <Icon name="download" /> {info.autoInstall ? 'Instalar o Sunshine' : 'Baixar o Sunshine'}
        </button>
      {:else if !info.running}
        <button class="btn primary" data-nav data-nav-default onclick={() => run(api.startRemotePlay, 'Sunshine ligado')}><Icon name="power" /> Ligar</button>
      {:else if info.started}
        <button class="btn" data-nav data-nav-default onclick={() => run(api.stopRemotePlay, 'Sunshine desligado')}><Icon name="power" /> Desligar</button>
      {/if}
    {/if}
  </header>

  <p class="muted">Transmita este console pra TV, celular ou outro PC da casa. Aqui roda o <strong>Sunshine</strong>; lá, o app <strong>Moonlight</strong>.</p>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else if !info}
    <p class="muted" aria-busy="true">Procurando o Sunshine…</p>
  {:else if !info.supported}
    <p class="note" role="note"><Icon name="info" /> {info.reason}</p>
  {:else}
    {#if task}
      <section class="progress" aria-label="Instalação">
        <p>{task.label}… {Math.round(task.progress * 100)}%</p>
        <div class="energy" role="progressbar" aria-label="Instalando o Sunshine" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(task.progress * 100)}>
          <span style:width={`${Math.floor(task.progress * 20) * 5}%`}></span>
        </div>
        <button class="btn small" data-nav data-nav-default onclick={() => cancelTask('remoteplay')}><Icon name="close" /> Cancelar</button>
      </section>
    {/if}

    <p class="state" class:on={info.running} data-testid="remoteplay-state">
      <Icon name={info.running ? 'check' : 'warn'} />
      {#if info.running}Ligado{info.started ? '' : ' (por fora do app: desligue por lá)'}
      {:else if info.installed}Instalado, desligado
      {:else}Não instalado{/if}
    </p>

    {#if info.reason}<p class="note" role="note"><Icon name="info" /> {info.reason}</p>{/if}

    {#if info.running}
      <div class="ip">
        <span class="muted small">Endereço deste console no Moonlight</span>
        <strong class="hud" data-testid="remoteplay-ip">{info.lanIP || 'sem rede'}</strong>
      </div>
      <ol class="steps">
        <li>No Moonlight, toque em <strong>+</strong> e digite <strong>{info.lanIP || 'o IP deste PC'}</strong> (ou escolha este PC na lista).</li>
        <li>O Moonlight mostra um PIN de 4 números.</li>
        <li>
          Abra a página do Sunshine, entre (no primeiro acesso, crie usuário e senha) e digite o PIN na aba <strong>PIN</strong>.
        </li>
      </ol>
      <div class="acts">
        <button class="btn" data-nav onclick={() => open(info!.pairUrl, api.openRemotePlayPage, 'Abri a página do Sunshine')}>
          <Icon name="lock" /> Abrir a página de parear
        </button>
        <span class="muted small url">{info.pairUrl}</span>
      </div>
      <p class="muted small">O navegador avisa "conexão não segura": é o certificado do próprio Sunshine, pode seguir.</p>
    {:else if !info.installed}
      <p class="muted small">
        {#if info.autoInstall}Instala pelo Flathub, só pro seu usuário. No fim, o sistema pede sua senha pra liberar o controle do outro lado.
        {:else}Abre a página oficial do Sunshine (LizardByte): rode o instalador e volte aqui.{/if}
      </p>
    {/if}
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  .state {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    font-weight: 700;
    margin: 0;
  }
  .state :global(svg) {
    color: #ff6b6b;
  }
  .state.on :global(svg) {
    color: #5dc44f;
  }
  .ip {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
    padding: 0.8rem 1rem;
    border: 2px solid color-mix(in srgb, var(--accent-2) 45%, transparent);
  }
  .ip strong {
    font-size: 1.8rem;
    overflow-wrap: anywhere;
  }
  .steps {
    margin: 0;
    padding-left: 1.4rem;
    display: grid;
    gap: 0.4rem;
  }
  .acts {
    display: flex;
    flex-wrap: wrap;
    gap: 0.8rem;
    align-items: center;
  }
  .url {
    overflow-wrap: anywhere;
  }
  .small {
    font-size: 0.85rem;
  }
  .progress {
    display: grid;
    gap: 0.5rem;
    justify-items: start;
  }
  .progress p {
    margin: 0;
  }
  .energy {
    width: 100%;
  }
  .note {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    color: var(--focus);
  }
</style>
