<script lang="ts">
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { play } from '../../lib/sound'
  import { app, confirm, toast } from '../../lib/state.svelte'
  import type { EmulatorPaths, SaveDirs, Settings } from '../../lib/types'

  let draft = $state<Settings>($state.snapshot(app.settings!) as Settings)
  let busy = $state(false)
  const dirty = $derived(JSON.stringify(draft) !== JSON.stringify(app.settings))

  function addDir() {
    draft.libraryDirs.push({ path: '', label: 'Nova pasta', writable: false })
  }

  async function save() {
    busy = true
    try {
      draft.libraryDirs = draft.libraryDirs.filter((d) => d.path.trim())
      app.settings = await api.saveSettings($state.snapshot(draft) as Settings)
      draft = $state.snapshot(app.settings) as Settings
      app.games = await api.refresh()
      toast(`Salvo. ${app.games.length} jogos na biblioteca.`, 'ok')
      play('accept')
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function rescan() {
    busy = true
    try {
      app.games = await api.refresh()
      toast(`${app.games.length} jogos encontrados`, 'ok')
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      busy = false
    }
  }

  async function toggleCovers() {
    const enabled = !draft.covers.enabled
    draft.covers.enabled = enabled
    try {
      app.settings = await api.saveSettings({ ...(app.settings as Settings), covers: { ...app.settings!.covers, enabled } })
      play('toggle')
      if (enabled) await api.updateCovers(false)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function updateCovers(all: boolean) {
    try {
      await api.updateCovers(all)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function reset() {
    if (!(await confirm('Voltar todas as configurações ao padrão? Pastas, acessibilidade e emuladores.', 'Restaurar'))) return
    app.settings = await api.resetSettings()
    draft = $state.snapshot(app.settings) as Settings
    app.games = await api.refresh()
    toast('Configurações restauradas', 'ok')
  }
</script>

<form class="panel-body" onsubmit={(e) => (e.preventDefault(), save())}>
  <h2>Biblioteca</h2>
  <p class="muted">Cada pasta tem uma subpasta por console: <code>snes/</code>, <code>psx/</code>, <code>ps2/</code>... (os mesmos nomes do ES-DE).</p>

  <fieldset>
    <legend>Pastas de jogos</legend>
    <ul class="dirs">
      {#each draft.libraryDirs as dir, i (i)}
        <li>
          <label class="sr-only" for={`dir-${i}`}>Pasta {i + 1}</label>
          <input id={`dir-${i}`} type="text" data-nav bind:value={dir.path} spellcheck="false" />
          <button
            type="button"
            class="btn small"
            data-nav
            aria-label={`Remover pasta ${dir.path}`}
            disabled={draft.libraryDirs.length === 1}
            onclick={() => draft.libraryDirs.splice(i, 1)}
          >
            <Icon name="trash" />
          </button>
        </li>
      {/each}
    </ul>
    <button type="button" class="btn small" data-nav onclick={addDir}><Icon name="plus" /> Adicionar pasta</button>
  </fieldset>

  <div class="field">
    <label for="import-dir">Pasta onde os jogos importados ficam</label>
    <input id="import-dir" type="text" data-nav bind:value={draft.importDir} spellcheck="false" />
  </div>

  <div class="field">
    <label for="esde-dir">Pasta do ES-DE (capas, descrições e anos vêm dos gamelists e da mídia de lá)</label>
    <input id="esde-dir" type="text" data-nav bind:value={draft.esdeDir} spellcheck="false" />
  </div>

  <details>
    <summary data-nav>Emuladores e saves (avançado)</summary>
    <p class="muted">
      {app.platform === 'linux'
        ? 'Vazio = os Flatpaks (make emulators instala).'
        : 'Vazio = procura nos lugares onde os instaladores oficiais põem (retroarch.com, ppsspp.org, pcsx2.net).'}
    </p>
    {#each [{ id: 'retroarch', label: 'RetroArch (executável)' }, { id: 'cores', label: 'Pasta dos cores do RetroArch' }, { id: 'ppsspp', label: 'PPSSPP (executável)' }, { id: 'pcsx2', label: 'PCSX2 (executável)' }] as f (f.id)}
      <div class="field">
        <label for={`emu-${f.id}`}>{f.label}</label>
        <input id={`emu-${f.id}`} type="text" data-nav bind:value={draft.emulators[f.id as keyof EmulatorPaths]} placeholder="padrão" spellcheck="false" />
      </div>
    {/each}
    {#each [{ id: 'retroarchSaves', label: 'Saves do RetroArch' }, { id: 'retroarchStates', label: 'Save states do RetroArch' }, { id: 'ppssppSaveData', label: 'Saves do PPSSPP (SAVEDATA)' }, { id: 'ppssppStates', label: 'Save states do PPSSPP' }, { id: 'pcsx2Memcards', label: 'Memory cards do PCSX2' }, { id: 'pcsx2States', label: 'Save states do PCSX2' }] as f (f.id)}
      <div class="field">
        <label for={`save-${f.id}`}>{f.label}</label>
        <input id={`save-${f.id}`} type="text" data-nav bind:value={draft.saves[f.id as keyof SaveDirs]} spellcheck="false" />
      </div>
    {/each}
  </details>

  <fieldset>
    <legend>Capas</legend>
    <button type="button" class="switch-row btn" data-nav role="switch" aria-checked={draft.covers.enabled} aria-describedby="covers-help" onclick={toggleCovers}>
      Baixar capas automaticamente: {draft.covers.enabled ? 'sim' : 'não'}
    </button>
    <p id="covers-help" class="muted">
      Capa, print e tela de título do acervo aberto do RetroArch (thumbnails.libretro.com), sem conta. O jogo é reconhecido pelo
      conteúdo da ROM ou pelo código do disco. Precisa de internet só pra baixar; depois fica tudo no PC. Capa sua em
      <code>covers/</code> ou do ES-DE tem prioridade.
    </p>
    <div class="actions">
      <button type="button" class="btn small" data-nav disabled={!draft.covers.enabled} onclick={() => updateCovers(false)}>Atualizar capas agora</button>
      <button type="button" class="btn small" data-nav disabled={!draft.covers.enabled} onclick={() => updateCovers(true)}>Tentar de novo as que faltam</button>
    </div>
  </fieldset>

  <button
    type="button"
    class="switch-row btn"
    data-nav
    role="switch"
    aria-checked={draft.fullscreen}
    onclick={() => (draft.fullscreen = !draft.fullscreen)}
  >
    Abrir jogos em tela cheia: {draft.fullscreen ? 'sim' : 'não'}
  </button>

  <div class="actions">
    <button type="submit" class="btn primary" data-nav disabled={busy || !dirty}><Icon name="check" /> Salvar</button>
    <button type="button" class="btn" data-nav disabled={busy} onclick={rescan}><Icon name="resume" /> Procurar jogos de novo</button>
    <button type="button" class="btn" data-nav disabled={busy} onclick={reset}>Restaurar padrão</button>
  </div>
</form>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.3rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  code {
    color: var(--accent-2);
  }
  fieldset {
    border: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    align-items: flex-start;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.6rem;
  }
  .dirs {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
    width: 100%;
  }
  .dirs li {
    display: flex;
    gap: 0.6rem;
  }
  .field {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }
  .actions {
    display: flex;
    gap: 0.9rem;
    flex-wrap: wrap;
  }
  details {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  summary {
    cursor: pointer;
    font-weight: 700;
    padding: 0.3rem;
    width: fit-content;
  }
  details[open] summary {
    margin-bottom: 0.8rem;
  }
  .btn:disabled {
    opacity: 0.45;
    cursor: not-allowed;
  }
  .switch-row {
    width: fit-content;
  }
</style>
