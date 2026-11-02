<script lang="ts">
  // Criar ou editar perfil: nome, avatar, cor, PIN e apagar.
  import { tick } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault, focusWithin } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { avatarName } from '../lib/avatars'
  import { app, askPIN, back, confirm, prompt, resetTo, toast } from '../lib/state.svelte'

  // setup: primeira vez do app (configurar o perfil principal antes de tudo)
  let { id, setup = false, suggest = '', embedded = false, onclose }: { id?: string; setup?: boolean; suggest?: string; embedded?: boolean; onclose?: () => void } = $props()

  const COLOR_NAMES = ['Vermelho', 'Laranja', 'Amarelo', 'Verde', 'Azul', 'Roxo', 'Rosa', 'Turquesa']

  const avatars = $derived(app.profile?.avatars ?? [])
  const colors = $derived(app.profile?.colors ?? [])
  const existing = $derived(id ? app.profile?.profiles.find((p) => p.id === id) : undefined)
  let name = $state('')
  let avatar = $state('')
  let color = $state('')
  let hasPIN = $state(false)
  let kids = $state(false)
  let daily = $state(0)
  let bedtime = $state('')
  const DAILY = [
    { v: 0, label: 'Sem limite' },
    { v: 30, label: '30 min' },
    { v: 60, label: '1 h' },
    { v: 120, label: '2 h' },
    { v: 180, label: '3 h' },
  ]
  const BEDTIMES = ['', '20:00', '21:00', '22:00']

  $effect(() => {
    const p = existing
    const used = new Set(app.profile?.profiles.map((x) => x.color))
    name = setup ? suggest : (p?.name ?? '')
    avatar = p?.avatar ?? avatars[(app.profile?.profiles.length ?? 0) % avatars.length] ?? 'knight'
    color = p?.color ?? colors.find((c) => !used.has(c)) ?? colors[0] ?? '#3a86d8'
    hasPIN = !!p?.hasPIN
    kids = !!p?.kids
    daily = p?.dailyMinutes ?? 0
    bedtime = p?.bedtime ?? ''
  })

  async function setLimits(minutes: number, bed: string) {
    if (!existing) return
    try {
      const p = await api.setProfileLimits(existing.id, minutes, bed)
      daily = p.dailyMinutes
      bedtime = p.bedtime
      await refresh()
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  async function toggleKids() {
    if (!existing) return
    try {
      const p = await api.setProfileKids(existing.id, !kids)
      kids = p.kids
      await refresh()
      play('toggle')
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    if (!embedded) tick().then(focusDefault)
  })

  $effect(() =>
    pushHandler((a) => {
      if (app.pin || app.prompt || app.confirm) return false
      if (setup && (a === 'back' || a === 'menu' || a === 'search')) return true // primeiro configura
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'random'
    }),
  )

  // embutida: concluir volta pra lista da tela Você; sozinha, volta de tela
  function close() {
    if (embedded) onclose?.()
    else back()
  }

  async function editName() {
    await prompt('Nome do perfil', name, 'Usar este nome', async (v) => {
      if (!v.trim()) return 'O perfil precisa de um nome'
      name = v.trim()
      return null
    }, { max: 20 })
    tick().then(() => (embedded ? focusWithin(root) : focusDefault()))
  }

  async function refresh() {
    app.profile = await api.profiles()
  }

  async function save() {
    try {
      if (existing) await api.updateProfile(existing.id, name, avatar, color)
      else await api.createProfile(name, avatar, color)
      await refresh()
      play('accept')
      if (setup) {
        toast(`Olá, ${name}!`, 'ok', 2500)
        resetTo({ name: 'home' })
        return
      }
      toast(existing ? 'Perfil salvo' : `Perfil "${name}" criado`, 'ok', 2500)
      close()
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  const check = (pid: string) => async (pin: string) => {
    try {
      await api.checkProfilePIN(pid, pin)
      return null
    } catch (err) {
      return (err as Error).message
    }
  }

  async function changePIN(remove: boolean) {
    if (!existing) return
    let current = ''
    if (hasPIN) {
      const got = await askPIN('PIN atual', check(existing.id))
      if (got === null) return
      current = got
    }
    let next = ''
    if (!remove) {
      const first = await askPIN('Novo PIN', async () => null)
      if (first === null) return
      const again = await askPIN('Repita o novo PIN', async (v) => (v === first ? null : 'Os PINs não batem'))
      if (again === null) return
      next = first
    }
    try {
      const p = await api.setProfilePIN(existing.id, current, next)
      hasPIN = p.hasPIN
      await refresh()
      toast(remove ? 'PIN removido' : 'PIN definido', 'ok', 2500)
    } catch (err) {
      toast((err as Error).message, 'error')
    }
    tick().then(() => (embedded ? focusWithin(root) : focusDefault()))
  }

  async function remove() {
    if (!existing) return
    const ok = await confirm(
      `Apagar o perfil "${existing.name}"? Favoritos, listas, histórico e saves do RetroArch dele saem do app (a pasta fica guardada em profiles/apagados).`,
      'Apagar perfil',
      true,
    )
    if (!ok) return
    let pin = ''
    if (hasPIN) {
      const got = await askPIN(`PIN de ${existing.name}`, check(existing.id))
      if (got === null) return
      pin = got
    }
    try {
      await api.deleteProfile(existing.id, pin)
      await refresh()
      toast('Perfil apagado', 'ok', 2500)
      close()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
  let root = $state<HTMLElement | null>(null)
</script>

<div class="page" class:embedded bind:this={root}>
  {#if !embedded}<TopBar actions={false} />{/if}
  <main class="editor" aria-labelledby={embedded ? undefined : 'editor-title'}>
    <div class="preview">
      <Avatar {avatar} {color} size="10rem" />
      <p class="pixel big-name">{name || 'Sem nome'}</p>
    </div>
    <div class="form">
      {#if !embedded}<h1 id="editor-title" class="pixel">{setup ? 'Boas-vindas! Monte seu perfil' : existing ? 'Editar perfil' : 'Novo perfil'}</h1>{/if}
      {#if setup}<p class="muted">Escolha seu nome e um personagem. Dá pra mudar depois e criar perfis pra mais gente.</p>{/if}
      <button class="btn field" data-nav data-nav-default onclick={editName}>
        <span class="muted">Nome</span> <strong>{name || 'Escolher nome'}</strong>
      </button>

      <fieldset>
        <legend>Avatar</legend>
        <div class="choices" role="radiogroup" aria-label="Avatar">
          {#each avatars as a (a)}
            <button class="choice" data-nav role="radio" aria-checked={avatar === a} aria-label={avatarName(a)} onclick={() => (avatar = a)}>
              <Avatar avatar={a} color={color} size="3.6rem" />
            </button>
          {/each}
        </div>
      </fieldset>

      <fieldset>
        <legend>Cor</legend>
        <div class="choices" role="radiogroup" aria-label="Cor">
          {#each colors as c, i (c)}
            <button class="choice swatch" data-nav role="radio" aria-checked={color === c} aria-label={COLOR_NAMES[i] ?? c} onclick={() => (color = c)}>
              <span style:background={c}></span>
            </button>
          {/each}
        </div>
      </fieldset>

      {#if existing && !setup}
        <fieldset>
          <legend>PIN</legend>
          <p class="muted">Com PIN, só quem sabe os 4 números entra neste perfil.</p>
          <div class="row">
            <button class="btn" data-nav onclick={() => changePIN(false)}><Icon name="lock" /> {hasPIN ? 'Trocar PIN' : 'Definir PIN'}</button>
            {#if hasPIN}<button class="btn" data-nav onclick={() => changePIN(true)}>Remover PIN</button>{/if}
          </div>
        </fieldset>
      {/if}

      {#if existing && !existing.main && existing.id !== app.profile?.current}
        <fieldset>
          <legend>Infantil</legend>
          <button class="btn" data-nav role="switch" aria-checked={kids} aria-describedby="kids-help" onclick={toggleKids}>
            <Icon name="star" /> Perfil infantil: {kids ? 'sim' : 'não'}
          </button>
          <p class="muted" id="kids-help">
            Só jogos liberados (no editor de cada jogo), botões maiores e nada de configurações, importar, apagar saves
            ou editar. Pra sair, o PIN do perfil adulto (defina um).
          </p>
          {#if kids}
            <p class="limit-label" id="daily-label">Tempo por dia</p>
            <div class="segmented" role="group" aria-labelledby="daily-label">
              {#each DAILY as d (d.v)}
                <button class="btn small" data-nav aria-pressed={daily === d.v} onclick={() => setLimits(d.v, bedtime)}>{d.label}</button>
              {/each}
            </div>
            <p class="limit-label" id="bed-label">Hora de dormir</p>
            <div class="segmented" role="group" aria-labelledby="bed-label">
              {#each BEDTIMES as b (b)}
                <button class="btn small" data-nav aria-pressed={bedtime === b} onclick={() => setLimits(daily, b)}>{b || 'Sem'}</button>
              {/each}
            </div>
          {/if}
        </fieldset>
      {/if}

      <div class="row">
        <button class="btn primary" data-nav disabled={!name} onclick={save}><Icon name="check" /> {setup ? 'Começar' : existing ? 'Salvar' : 'Criar perfil'}</button>
        {#if existing && !existing.main && existing.id !== app.profile?.current}
          <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar perfil</button>
        {/if}
      </div>
    </div>
  </main>
  {#if !embedded}<HelpBar hints={setup ? [{ button: 'south', label: 'Escolher' }] : [{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />{/if}
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
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .editor {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    display: grid;
    grid-template-columns: minmax(12rem, 18rem) 1fr;
    gap: 3rem;
    padding: 1rem 3rem 2rem;
    align-items: start;
  }
  .preview {
    display: grid;
    justify-items: center;
    gap: 1rem;
    padding-top: 3rem;
  }
  .big-name {
    font-size: 1.3rem;
    text-align: center;
    overflow-wrap: anywhere;
  }
  .form {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
    max-width: 44rem;
  }
  h1 {
    font-size: clamp(1.6rem, 3vw, 2.4rem);
  }
  .field {
    justify-content: flex-start;
    gap: 1rem;
  }
  fieldset {
    border: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.4rem;
  }
  .choices {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
    padding: 0.3rem;
  }
  .choice {
    padding: 0.25rem;
    background: none;
    border: 3px solid transparent;
    cursor: pointer;
  }
  .choice[aria-checked='true'] {
    border-color: var(--accent);
  }
  .choice:focus-visible {
    outline: 3px dashed var(--text);
    outline-offset: 2px;
  }
  .swatch span {
    display: block;
    width: 2.6rem;
    height: 2.6rem;
    border: 3px solid rgba(0, 0, 0, 0.35);
  }
  .limit-label {
    font-weight: 700;
    margin: 0.4rem 0 0;
  }
  .segmented {
    display: flex;
    flex-wrap: wrap;
    gap: 0.6rem;
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.9rem;
  }
  @media (max-width: 760px) {
    .editor {
      grid-template-columns: 1fr;
    }
  }
</style>
