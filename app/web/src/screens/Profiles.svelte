<script lang="ts" module>
  // o modo gerenciar sobrevive à ida e volta do editor
  let keepManaging = false
</script>

<script lang="ts">
  // "Quem vai jogar?": escolher, criar e (no modo gerenciar) editar perfis.
  import { tick } from 'svelte'
  import Avatar from '../components/Avatar.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon from '../components/Icon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, applyBoot, askPIN, back, go, isKids, resetTo, toast } from '../lib/state.svelte'
  import type { Profile } from '../lib/types'

  const MAX = 8
  let managing = $state(keepManaging)
  $effect(() => {
    keepManaging = managing
  })
  const list = $derived(app.profile?.profiles ?? [])
  const canGoBack = $derived(app.stack.length > 1)

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    tick().then(focusDefault)
    // a lista pode ter mudado no editor
    api.profiles().then((p) => (app.profile = p)).catch(() => {})
  })

  $effect(() =>
    pushHandler((a) => {
      if (app.pin || app.confirm) return false
      if (a === 'back') {
        if (managing) {
          managing = false
          return true
        }
        return !canGoBack // na abertura, escolher é obrigatório
      }
      return a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab' || a === 'search' || a === 'random' || a === 'menu'
    }),
  )

  async function enter(p: Profile) {
    if (managing) {
      go({ name: 'profile', id: p.id })
      return
    }
    let pin = ''
    if (p.hasPIN) {
      const got = await askPIN(`PIN de ${p.name}`, async (v) => {
        try {
          await api.checkProfilePIN(p.id, v)
          return null
        } catch (err) {
          return (err as Error).message
        }
      })
      if (got === null) return
      pin = got
    }
    try {
      const boot = await api.selectProfile(p.id, pin)
      applyBoot(boot)
      resetTo({ name: 'home' })
      play('start')
      toast(`Olá, ${p.name}!`, 'ok', 2500)
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="page">
  <main class="who" aria-labelledby="who-title">
    <h1 id="who-title" class="pixel">{managing ? 'Gerenciar perfis' : 'Quem vai jogar?'}</h1>
    <ul class="cards">
      {#each list as p (p.id)}
        <li>
          <button
            class="card"
            data-nav
            data-profile={p.id}
            data-nav-default={p.id === app.profile?.current ? '' : undefined}
            onclick={() => enter(p)}
          >
            <span class="art">
              <Avatar avatar={p.avatar} color={p.color} size="8rem" />
              {#if managing}<span class="edit" aria-hidden="true"><Icon name="pencil" /></span>{/if}
            </span>
            <span class="name">
              {#if managing}<span class="sr-only">Editar </span>{/if}{p.name}
            </span>
            {#if p.hasPIN}<span class="tag"><Icon name="lock" size="0.9em" /> PIN</span>{/if}
            {#if p.kids}<span class="tag kids">Infantil</span>{/if}
          </button>
        </li>
      {/each}
      {#if list.length < MAX && !isKids()}
        <li>
          <button class="card add" data-nav onclick={() => go({ name: 'profile' })}>
            <span class="art plus" aria-hidden="true"><Icon name="plus" size="3rem" /></span>
            <span class="name">Adicionar perfil</span>
          </button>
        </li>
      {/if}
    </ul>
    <div class="actions">
      {#if !isKids()}
        <button class="btn" data-nav aria-pressed={managing} onclick={() => (managing = !managing)}>
          <Icon name="sliders" /> {managing ? 'Pronto' : 'Gerenciar perfis'}
        </button>
      {/if}
      {#if canGoBack}
        <button class="btn" data-nav onclick={() => { managing = false; keepManaging = false; back() }}><Icon name="back" /> Voltar</button>
      {/if}
    </div>
  </main>
  <HelpBar hints={[{ button: 'south', label: managing ? 'Editar' : 'Entrar' }, ...(canGoBack || managing ? [{ button: 'east', label: 'Voltar' }] : [])]} />
</div>

<style>
  .page {
    height: 100%;
    display: flex;
    flex-direction: column;
  }
  .who {
    flex: 1;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 2.4rem;
    padding: 2rem;
    overflow-y: auto;
  }
  h1 {
    font-size: clamp(1.8rem, 4vw, 3rem);
    text-align: center;
  }
  .cards {
    list-style: none;
    margin: 0;
    padding: 0.5rem;
    display: flex;
    flex-wrap: wrap;
    justify-content: center;
    gap: 1.6rem;
  }
  .card {
    display: grid;
    justify-items: center;
    gap: 0.6rem;
    padding: 0.8rem;
    width: 10.5rem;
    font: inherit;
    color: var(--text);
    background: none;
    border: 3px solid transparent;
    cursor: pointer;
  }
  .card:focus-visible,
  .card:hover {
    border-color: var(--accent);
  }
  .art {
    position: relative;
    display: grid;
    place-items: center;
  }
  .plus {
    width: 8rem;
    height: 8rem;
    border: 3px dashed color-mix(in srgb, var(--text) 50%, transparent);
  }
  .edit {
    position: absolute;
    inset: 0;
    display: grid;
    place-items: center;
    font-size: 2rem;
    background: rgba(0, 0, 0, 0.55);
    color: #fff;
  }
  .name {
    font-weight: 700;
    font-size: 1.1rem;
    text-align: center;
  }
  .tag {
    font-size: 0.8rem;
    padding: 0.1rem 0.5rem;
    border: 2px solid currentColor;
    color: var(--muted);
  }
  .tag.kids {
    color: var(--accent);
  }
  .actions {
    display: flex;
    gap: 1rem;
  }
</style>
