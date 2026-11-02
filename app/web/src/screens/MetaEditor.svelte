<script lang="ts">
  // Editor de metadados: cada campo mostra se é editado ou automático. A edição
  // vale por cima do ES-DE/libretro; restaurar volta ao automático.
  import { tick } from 'svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, back, confirm, game, prompt, putGame, system, toast } from '../lib/state.svelte'
  import type { MetaEdit, MetaInfo } from '../lib/types'

  let { id }: { id: string } = $props()
  const g = $derived(game(id))
  let info = $state<MetaInfo | null>(null)

  $effect(() => {
    api
      .metaDetails(id)
      .then((m) => {
        info = m
        tick().then(focusDefault)
      })
      .catch((err) => toast((err as Error).message, 'error'))
  })
  $effect(() => {
    const sys = g ? system(g.system) : undefined
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : (sys?.palette ?? homePalette))
  })

  type TextKey = 'name' | 'developer' | 'year' | 'players' | 'genres' | 'summary'
  const fields: { key: TextKey; label: string; max: number; multiline?: boolean; hint?: string }[] = [
    { key: 'name', label: 'Nome', max: 120 },
    { key: 'developer', label: 'Desenvolvedora', max: 80 },
    { key: 'year', label: 'Ano', max: 4, hint: 'ex.: 1990' },
    { key: 'players', label: 'Jogadores', max: 20, hint: 'ex.: 1-2' },
    { key: 'genres', label: 'Gêneros', max: 200, hint: 'separados por vírgula' },
    { key: 'summary', label: 'Descrição', max: 4000, multiline: true },
  ]

  function edited(key: TextKey): boolean {
    return info?.edit[key] !== undefined && info?.edit[key] !== null
  }

  function value(key: TextKey): string {
    if (!info) return ''
    const src = edited(key) ? info.edit : info.auto
    const v = src[key as keyof typeof src]
    if (key === 'genres') return ((v as string[] | null) ?? []).join(', ')
    if (key === 'year') return v ? String(v) : ''
    return (v as string) ?? ''
  }

  async function save(next: MetaEdit): Promise<string | null> {
    try {
      putGame(await api.saveMeta(id, next))
      info = await api.metaDetails(id)
      play('accept')
      return null
    } catch (err) {
      return (err as Error).message
    }
  }

  function parse(key: TextKey, raw: string): MetaEdit[keyof MetaEdit] | Error {
    const v = raw.trim()
    if (key === 'year') {
      if (v === '') return 0
      if (!/^\d{4}$/.test(v)) return new Error('ano com 4 dígitos (ex.: 1990)')
      return Number(v)
    }
    if (key === 'genres') return v.split(',').map((x) => x.trim()).filter(Boolean)
    return v
  }

  async function editField(f: (typeof fields)[number]) {
    if (!info) return
    await prompt(`${f.label}${f.hint ? ` (${f.hint})` : ''}`, value(f.key), 'Salvar', async (raw) => {
      const parsed = parse(f.key, raw)
      if (parsed instanceof Error) return parsed.message
      return save({ ...info!.edit, [f.key]: parsed })
    }, { max: f.max, multiline: f.multiline })
  }

  async function resetField(key: TextKey) {
    if (!info) return
    const next = { ...info.edit }
    delete next[key]
    const err = await save(next)
    if (err) toast(err, 'error')
  }

  async function set<K extends 'rating' | 'completed' | 'broken' | 'kidsOk'>(key: K, v: MetaEdit[K]) {
    if (!info) return
    const err = await save({ ...info.edit, [key]: v })
    if (err) toast(err, 'error')
  }

  async function chooseCover() {
    if (!info) return
    if (app.mode === 'desktop') {
      const files = await api.pickFiles().catch(() => null)
      if (files?.[0]) {
        const err = await save({ ...info.edit, cover: files[0] })
        if (err) toast(err, 'error')
      }
      return
    }
    await prompt('Caminho da imagem da capa', info.edit.cover ?? '', 'Usar esta capa', async (path) => save({ ...info!.edit, cover: path.trim() }), { max: 400 })
  }

  async function clearCover() {
    if (!info) return
    const next = { ...info.edit }
    delete next.cover
    const err = await save(next)
    if (err) toast(err, 'error')
  }

  // busca na internet (Wikidata/Wikipedia): só preenche o que está vazio
  let fetching = $state(false)
  let source = $state('')
  const canFetch = $derived(!!info && (!value('summary') || !value('year') || !value('genres')))

  async function fetchWeb() {
    if (!info || fetching) return
    fetching = true
    try {
      const r = await api.fetchMeta(id)
      const next = { ...info.edit }
      const filled: string[] = []
      if (!value('summary') && r.summary) (next.summary = r.summary), filled.push('descrição')
      if (!value('year') && r.year) (next.year = r.year), filled.push('ano')
      if (!value('genres') && r.genres.length) (next.genres = r.genres), filled.push('gêneros')
      if (!filled.length) {
        toast('A internet não tem nada novo pra este jogo', 'info', 3500)
        return
      }
      const err = await save(next)
      if (err) return toast(err, 'error')
      source = r.source
      toast(`Preenchido: ${filled.join(', ')}`, 'ok', 3500)
    } catch (err) {
      toast((err as Error).message, 'error')
    } finally {
      fetching = false
    }
  }

  async function resetAll() {
    if (!(await confirm('Voltar todos os dados deste jogo pro automático?', 'Restaurar tudo'))) return
    try {
      putGame(await api.resetMeta(id))
      info = await api.metaDetails(id)
      toast('Dados restaurados', 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="body">
    <h1 class="pixel">Editar: {g?.name ?? ''}</h1>
    {#if info}
      <ul class="fields">
        {#each fields as f, i (f.key)}
          <li class="field">
            <button class="value" data-nav data-nav-default={i === 0 ? '' : undefined} onclick={() => editField(f)}>
              <span class="label">{f.label}</span>
              <span class="text" class:empty={!value(f.key)}>{value(f.key) || 'vazio'}</span>
              <span class="tag" class:on={edited(f.key)}>{edited(f.key) ? 'editado' : 'automático'}</span>
            </button>
            {#if edited(f.key)}
              <button class="btn small" data-nav aria-label={`Restaurar ${f.label.toLowerCase()} automático`} onclick={() => resetField(f.key)}>
                <Icon name="resume" /> Restaurar
              </button>
            {/if}
          </li>
        {/each}
      </ul>

      {#if canFetch || source}
        <div class="web">
          {#if canFetch}
            <button class="btn" data-nav disabled={fetching} aria-describedby="web-help" onclick={fetchWeb}>
              <Icon name="globe" /> {fetching ? 'Buscando…' : 'Buscar na internet'}
            </button>
            <span id="web-help" class="muted">Descrição, ano e gêneros da Wikipedia, só nos campos vazios.</span>
          {/if}
          {#if source}
            <p class="muted" role="status">Fonte: {source.replace(/^https?:\/\//, '')}</p>
          {/if}
        </div>
      {/if}

      <fieldset>
        <legend>Nota</legend>
        <div class="stars" role="radiogroup" aria-label="Nota">
          {#each [0, 1, 2, 3, 4, 5] as n (n)}
            <button class="star" role="radio" aria-checked={info.edit.rating === n} data-nav onclick={() => set('rating', n)}>
              {n === 0 ? 'Sem nota' : '★'.repeat(n)}
            </button>
          {/each}
        </div>
      </fieldset>

      <div class="switches">
        <button class="btn" data-nav role="switch" aria-checked={info.edit.completed} onclick={() => set('completed', !info!.edit.completed)}>
          <Icon name="check" /> Completo: {info.edit.completed ? 'sim' : 'não'}
        </button>
        <button class="btn" data-nav role="switch" aria-checked={info.edit.broken} onclick={() => set('broken', !info!.edit.broken)}>
          <Icon name="warn" /> Quebrado (não abre direito): {info.edit.broken ? 'sim' : 'não'}
        </button>
        <button class="btn" data-nav role="switch" aria-checked={!!info.edit.kidsOk} onclick={() => set('kidsOk', !info!.edit.kidsOk)}>
          <Icon name="star" /> Liberado no perfil infantil: {info.edit.kidsOk ? 'sim' : 'não'}
        </button>
      </div>

      <fieldset>
        <legend>Capa</legend>
        <p class="muted">{info.edit.cover ? `Escolhida: ${info.edit.cover}` : info.autoCover ? 'Automática (ES-DE, pasta covers/ ou libretro)' : 'Sem capa: o app desenha a mídia do console'}</p>
        <div class="switches">
          <button class="btn" data-nav onclick={chooseCover}><Icon name="folder" /> Escolher imagem</button>
          {#if info.edit.cover}
            <button class="btn" data-nav onclick={clearCover}>Usar capa automática</button>
          {/if}
        </div>
      </fieldset>

      <div class="switches end">
        <button class="btn" data-nav onclick={resetAll}>Restaurar tudo</button>
        <button class="btn primary" data-nav onclick={() => back()}><Icon name="check" /> Pronto</button>
      </div>
    {/if}
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Editar' }, { button: 'east', label: 'Voltar' }]} />
</div>

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 1.5rem 3.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.3rem;
    max-width: 64rem;
  }
  h1 {
    font-size: 2rem;
  }
  .fields {
    list-style: none;
    padding: 0.3rem;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  .field {
    display: flex;
    gap: 0.6rem;
    align-items: center;
  }
  .value {
    flex: 1;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: #000;
    border: 2px solid color-mix(in srgb, var(--accent-2) 70%, transparent);
    padding: 0.6rem 0.9rem;
    display: grid;
    grid-template-columns: 10rem 1fr auto;
    gap: 1rem;
    align-items: center;
    cursor: pointer;
    min-width: 0;
  }
  .label {
    font-weight: 700;
  }
  .text {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .text.empty {
    color: var(--muted);
    font-style: italic;
  }
  .tag {
    font-size: 0.8rem;
    color: var(--muted);
    border: 2px solid currentColor;
    padding: 0 0.4rem;
  }
  .tag.on {
    color: var(--accent);
  }
  fieldset {
    border: none;
    padding: 0;
    margin: 0;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.5rem;
  }
  .stars {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }
  .star {
    font: inherit;
    color: var(--text);
    background: var(--surface);
    border: 3px solid transparent;
    padding: 0.4rem 0.7rem;
    cursor: pointer;
  }
  .star[aria-checked='true'] {
    border-color: var(--accent);
    color: var(--accent);
  }
  .switches {
    display: flex;
    gap: 0.8rem;
    flex-wrap: wrap;
  }
  .web {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 0.8rem;
  }
  .web p {
    flex-basis: 100%;
    margin: 0;
    overflow-wrap: anywhere;
  }
  .switches.end {
    justify-content: flex-end;
  }
</style>
