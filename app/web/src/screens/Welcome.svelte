<script lang="ts">
  // Primeira vez do app: entrar no servidor RomM ou jogar offline. Depois vem
  // a configuração do perfil. A senha só vai pro login (o app guarda um token).
  import { tick } from 'svelte'
  import Icon from '../components/Icon.svelte'
  import { api } from '../lib/api'
  import { pushHandler } from '../lib/input'
  import { focusDefault } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, prompt, resetTo } from '../lib/state.svelte'

  const FACES = ['pac-man', 'mario', 'link', 'samus', 'mega-man', 'sonic', 'kirby', 'donkey-kong', 'pikachu', 'simon', 'fox', 'ryu', 'yoshi', 'snake', 'cloud', 'terry']

  let host = $state('')
  let user = $state('')
  let pass = $state('')
  let error = $state('')
  let busy = $state(false)

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
    tick().then(focusDefault)
  })

  // aqui não tem "voltar": escolhe servidor ou offline
  $effect(() =>
    pushHandler((a) => {
      if (app.prompt) return false
      return a === 'back' || a === 'menu' || a === 'search' || a === 'random' || a === 'fav' || a === 'details' || a === 'prevTab' || a === 'nextTab'
    }),
  )

  async function ask(title: string, value: string, secret = false): Promise<string | null> {
    let got: string | null = null
    await prompt(title, value, 'OK', async (v) => {
      got = secret ? v : v.trim()
      return null
    }, { max: secret ? 200 : 120, secret })
    tick().then(focusDefault)
    return got
  }

  function next(suggest = '') {
    resetTo({ name: 'profile', id: 'main', setup: true, suggest })
  }

  async function login() {
    if (!host || !user || !pass) {
      error = 'Preencha endereço, usuário e senha'
      play('error')
      return
    }
    busy = true
    error = ''
    try {
      const info = await api.connectRemote(host, user, pass)
      pass = ''
      play('accept')
      const name = info.user.charAt(0).toUpperCase() + info.user.slice(1)
      next(name)
    } catch (err) {
      error = (err as Error).message
      play('error')
    } finally {
      busy = false
    }
  }
</script>

<div class="welcome">
  <div class="faces" aria-hidden="true">
    {#each FACES as f, i (f)}
      <img src={`/avatars/${f}.png`} alt="" style:--i={i} />
    {/each}
  </div>
  <main class="card panel" aria-labelledby="welcome-title">
    <p class="logo pixel" aria-hidden="true">ARK<span>FLIX</span></p>
    <h1 id="welcome-title" class="pixel">Bem-vindo!</h1>
    <p class="muted lead">Entre no seu servidor RomM pra ver os jogos de lá e guardar os saves na nuvem, ou jogue só com o que está neste computador.</p>

    <form
      class="form"
      aria-label="Entrar no servidor RomM"
      onsubmit={(e) => {
        e.preventDefault()
        login()
      }}
    >
      <button type="button" class="field" data-nav data-nav-default onclick={async () => (host = (await ask('Endereço do servidor', host)) ?? host)}>
        <span class="label">Servidor</span>
        <span class="value" class:empty={!host}>{host || 'meu-romm.duckdns.org'}</span>
      </button>
      <button type="button" class="field" data-nav onclick={async () => (user = (await ask('Usuário do RomM', user)) ?? user)}>
        <span class="label">Usuário</span>
        <span class="value" class:empty={!user}>{user || 'seu usuário'}</span>
      </button>
      <button type="button" class="field" data-nav onclick={async () => (pass = (await ask('Senha do RomM', '', true)) ?? pass)}>
        <span class="label">Senha</span>
        <span class="value" class:empty={!pass}>{pass ? '•'.repeat(Math.min(pass.length, 12)) : 'sua senha'}</span>
      </button>
      {#if error}<p class="error" role="alert">{error}</p>{/if}
      <button type="submit" class="btn primary wide" data-nav disabled={busy}>
        <Icon name="cloud" /> {busy ? 'Entrando…' : 'Entrar com o servidor'}
      </button>
    </form>

    <div class="or" aria-hidden="true"><span>ou</span></div>
    <button class="btn wide" data-nav onclick={() => next()}><Icon name="play" /> Jogar offline</button>
    <p class="muted small">Dá pra conectar depois em Configurações › Servidor.</p>
  </main>
</div>

<style>
  .welcome {
    position: fixed;
    inset: 0;
    display: grid;
    place-items: center;
    overflow: hidden;
    background:
      radial-gradient(ellipse at 50% 35%, color-mix(in srgb, var(--accent) 22%, transparent), transparent 60%),
      var(--bg);
  }
  .faces {
    position: absolute;
    inset: 0;
    display: grid;
    grid-template-columns: repeat(8, 1fr);
    align-content: space-around;
    justify-items: center;
    opacity: 0.16;
    pointer-events: none;
  }
  .faces img {
    width: clamp(3rem, 7vw, 6rem);
    image-rendering: pixelated;
    animation: bob 3s steps(4) infinite;
    animation-delay: calc(var(--i) * -0.37s);
  }
  @keyframes bob {
    50% {
      transform: translateY(-0.6rem);
    }
  }
  .card {
    position: relative;
    width: min(30rem, calc(100vw - 2rem));
    display: grid;
    gap: 1rem;
    padding: 2rem 2.2rem;
    border-color: var(--accent);
    box-shadow: 0 0 14px color-mix(in srgb, var(--accent) 60%, transparent), 0 0 40px color-mix(in srgb, var(--accent) 25%, transparent);
  }
  .logo {
    font-size: 2.4rem;
    text-align: center;
    font-size-adjust: 0.42;
    text-shadow: 0 0 6px var(--accent), 0 0 18px var(--accent), 3px 3px 0 #000;
  }
  .logo span {
    color: var(--accent);
  }
  h1 {
    font-size: 1.5rem;
    text-align: center;
  }
  .lead {
    text-align: center;
  }
  .form {
    display: grid;
    gap: 0.7rem;
  }
  .field {
    display: grid;
    grid-template-columns: 6rem 1fr;
    align-items: center;
    gap: 0.8rem;
    padding: 0.75rem 1rem;
    font: inherit;
    color: var(--text);
    text-align: left;
    background: color-mix(in srgb, var(--bg) 70%, #000);
    border: 2px solid color-mix(in srgb, var(--accent-2) 70%, transparent);
    cursor: text;
  }
  .field:focus-visible,
  .field:hover {
    border-color: var(--focus);
  }
  .label {
    font-weight: 700;
    color: var(--muted);
  }
  .value {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .value.empty {
    color: var(--muted);
    font-style: italic;
  }
  .wide {
    width: 100%;
    justify-content: center;
  }
  .error {
    color: #ff9c9c;
    font-weight: 700;
  }
  .or {
    display: flex;
    align-items: center;
    gap: 0.8rem;
    color: var(--muted);
  }
  .or::before,
  .or::after {
    content: '';
    flex: 1;
    border-top: 2px dashed color-mix(in srgb, var(--text) 25%, transparent);
  }
  .small {
    font-size: 0.85rem;
    text-align: center;
  }
</style>
