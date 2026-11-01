<script lang="ts">
  // Criar ou editar perfil: nome, avatar, cor, PIN e apagar.
  import { tick } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, askPIN, back, confirm, prompt, toast } from '../lib/state.svelte'

  let { id }: { id?: string } = $props()

  const COLOR_NAMES = ['Vermelho', 'Laranja', 'Amarelo', 'Verde', 'Azul', 'Roxo', 'Rosa', 'Turquesa']
  const AVATAR_NAMES: Record<string, string> = {
    knight: 'Cavaleiro',
    mage: 'Mago',
    ghost: 'Fantasma',
    cat: 'Gato',
    robot: 'Robô',
    mushroom: 'Cogumelo',
    alien: 'Alienígena',
    dragon: 'Dragão',
  }

  const avatars = $derived(app.profile?.avatars ?? [])
  const colors = $derived(app.profile?.colors ?? [])
  const existing = $derived(id ? app.profile?.profiles.find((p) => p.id === id) : undefined)
  let name = $state('')
  let avatar = $state('')
  let color = $state('')
  let hasPIN = $state(false)

  $effect(() => {
    const p = existing
    const used = new Set(app.profile?.profiles.map((x) => x.color))
    name = p?.name ?? ''
    avatar = p?.avatar ?? avatars[(app.profile?.profiles.length ?? 0) % avatars.length] ?? 'knight'
    color = p?.color ?? colors.find((c) => !used.has(c)) ?? colors[0] ?? '#3a86d8'
    hasPIN = !!p?.hasPIN
  })

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    tick().then(focusDefault)
  })

  $effect(() =>
    pushHandler((a) => {
      if (app.pin || app.prompt || app.confirm) return false
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'random'
    }),
  )

  async function editName() {
    await prompt('Nome do perfil', name, 'Usar este nome', async (v) => {
      if (!v.trim()) return 'O perfil precisa de um nome'
      name = v.trim()
      return null
    }, { max: 20 })
    tick().then(focusDefault)
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
      toast(existing ? 'Perfil salvo' : `Perfil "${name}" criado`, 'ok', 2500)
      back()
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
    tick().then(focusDefault)
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
      back()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="editor" aria-labelledby="editor-title">
    <div class="preview">
      <Avatar {avatar} {color} size="10rem" />
      <p class="pixel big-name">{name || 'Sem nome'}</p>
    </div>
    <div class="form">
      <h1 id="editor-title" class="pixel">{existing ? 'Editar perfil' : 'Novo perfil'}</h1>
      <button class="btn field" data-nav data-nav-default onclick={editName}>
        <span class="muted">Nome</span> <strong>{name || 'Escolher nome'}</strong>
      </button>

      <fieldset>
        <legend>Avatar</legend>
        <div class="choices" role="radiogroup" aria-label="Avatar">
          {#each avatars as a (a)}
            <button class="choice" data-nav role="radio" aria-checked={avatar === a} aria-label={AVATAR_NAMES[a] ?? a} onclick={() => (avatar = a)}>
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

      {#if existing}
        <fieldset>
          <legend>PIN</legend>
          <p class="muted">Com PIN, só quem sabe os 4 números entra neste perfil.</p>
          <div class="row">
            <button class="btn" data-nav onclick={() => changePIN(false)}><Icon name="lock" /> {hasPIN ? 'Trocar PIN' : 'Definir PIN'}</button>
            {#if hasPIN}<button class="btn" data-nav onclick={() => changePIN(true)}>Remover PIN</button>{/if}
          </div>
        </fieldset>
      {/if}

      <div class="row">
        <button class="btn primary" data-nav disabled={!name} onclick={save}><Icon name="check" /> {existing ? 'Salvar' : 'Criar perfil'}</button>
        <button class="btn" data-nav onclick={() => back()}><Icon name="back" /> Voltar</button>
        {#if existing && !existing.main && existing.id !== app.profile?.current}
          <button class="btn danger" data-nav onclick={remove}><Icon name="trash" /> Apagar perfil</button>
        {/if}
      </div>
    </div>
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />
</div>

<style>
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
