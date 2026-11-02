<script lang="ts">
  // Emuladores: o que está instalado, e um botão que instala o que falta (ou
  // atualiza tudo). No Linux pelo Flathub; no Windows e no macOS, os
  // pacotes oficiais. O progresso vem pela tarefa "emulators".
  import Icon from '../../components/Icon.svelte'
  import { activity, cancelTask } from '../../lib/activity.svelte'
  import { api, on } from '../../lib/api'
  import { play } from '../../lib/sound'
  import { system, toast } from '../../lib/state.svelte'
  import type { EmulatorsInfo } from '../../lib/types'

  let info = $state<EmulatorsInfo | null>(null)
  let error = $state('')

  async function load() {
    try {
      info = await api.emulatorStatus()
      error = ''
    } catch (err) {
      error = (err as Error).message
    }
  }

  $effect(() => {
    load()
    return on<EmulatorsInfo>('emulators', (i) => (info = i))
  })

  const task = $derived(activity.tasks.find((t) => t.id === 'emulators' && t.state === 'running'))
  const missing = $derived(info ? info.emulators.filter((e) => !e.installed).length + info.cores.filter((c) => !c.installed).length : 0)
  const coresOk = $derived(info?.cores.filter((c) => c.installed).length ?? 0)
  const names = (slugs: string[]) => slugs.map((s) => system(s)?.short || s.toUpperCase()).join(', ')

  async function install(which: string[] = []) {
    try {
      await api.installEmulators(which)
      play('accept')
      toast(which.length ? 'Atualizando…' : 'Instalando os emuladores…', 'info', 2500, 'emulators')
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="panel-body">
  <header class="head">
    <h2>Emuladores</h2>
    {#if info?.canInstall && !task}
      {#if missing}
        <button class="btn primary" data-nav data-nav-default onclick={() => install()}><Icon name="download" /> Instalar o que falta ({missing})</button>
      {:else}
        <button class="btn" data-nav data-nav-default onclick={() => install(['retroarch', 'ppsspp', 'pcsx2', 'cores'])}><Icon name="resume" /> Atualizar tudo</button>
      {/if}
    {/if}
  </header>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else if !info}
    <p class="muted" aria-busy="true">Procurando…</p>
  {:else}
    {#if !info.canInstall && info.reason}
      <p class="note" role="note"><Icon name="info" /> {info.reason}</p>
    {/if}
    {#if task}
      <section class="progress" aria-label="Instalação">
        <p>{task.label}… {Math.round(task.progress * 100)}%</p>
        <div class="energy" role="progressbar" aria-label="Instalando emuladores" aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(task.progress * 100)}>
          <span style:width={`${Math.floor(task.progress * 20) * 5}%`}></span>
        </div>
        <button class="btn small" data-nav data-nav-default onclick={() => cancelTask('emulators')}><Icon name="close" /> Cancelar</button>
      </section>
    {/if}
    <ul class="emus">
      {#each info.emulators as e (e.id)}
        <li class:ok={e.installed}>
          <Icon name={e.installed ? 'check' : 'warn'} />
          <span class="text">
            <span class="label">{e.name}{#if e.version}<span class="muted"> · {e.version}</span>{/if}</span>
            <span class="muted small path">{e.installed ? e.path : 'não instalado'}</span>
          </span>
          {#if info.canInstall && !task}
            <button class="btn small" data-nav onclick={() => install([e.id])}>{e.installed ? 'Atualizar' : 'Instalar'}</button>
          {/if}
        </li>
      {/each}
      <li class:ok={coresOk === info.cores.length}>
        <Icon name={coresOk === info.cores.length ? 'check' : 'warn'} />
        <span class="text">
          <span class="label">Cores do RetroArch <span class="muted">· {coresOk} de {info.cores.length}</span></span>
          <span class="muted small">
            {#each info.cores as c, i (c.id)}{i ? ' · ' : ''}<span class:missing={!c.installed}>{c.id} ({names(c.systems)})</span>{/each}
          </span>
        </span>
        {#if info.canInstall && !task}
          <button class="btn small" data-nav onclick={() => install(['cores'])}>{coresOk ? 'Atualizar' : 'Instalar'}</button>
        {/if}
      </li>
    </ul>
    <p class="muted small">
      {#if info.platform === 'linux'}Instala pelo Flathub, só pro seu usuário. Os cores vêm do buildbot do libretro.
      {:else}Baixa os pacotes oficiais (RetroArch, PPSSPP e PCSX2) e os cores do buildbot do libretro.{/if}
      Configurações e saves ficam onde estão.
    </p>
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
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
  .emus {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .emus li {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.9rem;
    align-items: center;
    padding: 0.8rem 1rem;
    border: 2px solid color-mix(in srgb, #ff6b6b 60%, transparent);
  }
  .emus li.ok {
    border-color: color-mix(in srgb, var(--accent-2) 45%, transparent);
  }
  .emus li :global(svg) {
    color: #ff6b6b;
  }
  .emus li.ok :global(svg) {
    color: #5dc44f;
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    min-width: 0;
  }
  .label {
    font-weight: 700;
  }
  .small {
    font-size: 0.85rem;
  }
  .path {
    overflow-wrap: anywhere;
  }
  .missing {
    color: #ff9b9b;
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
