<script lang="ts">
  import type { Snippet } from 'svelte'
  import { unread } from '../lib/activity.svelte'
  import { versionArrow } from '../lib/update'
  import { app, back, currentProfile, glyphFamily, go, isKids } from '../lib/state.svelte'
  import { play } from '../lib/sound'
  import Avatar from './Avatar.svelte'
  import Glyph from './Glyph.svelte'
  import Icon from './Icon.svelte'

  let { children, actions = true }: { children?: Snippet; actions?: boolean } = $props()
  let now = $state(new Date())
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 15000)
    return () => clearInterval(t)
  })
  const me = $derived(currentProfile())
  const news = $derived(unread())
  const clock = $derived(now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }))
  // "Seg, 02 de nov" (só a primeira letra maiúscula)
  const date = $derived.by(() => {
    const d = now.toLocaleDateString('pt-BR', { weekday: 'short', day: '2-digit', month: 'short' }).replace(/\./g, '')
    return d.charAt(0).toUpperCase() + d.slice(1)
  })
  const canBack = $derived(app.stack.length > 1)
  // voltando pro Início, o botão diz isso (fica claro onde ele leva)
  const toHome = $derived(app.stack.length > 1 && app.stack[app.stack.length - 2].name === 'home')
  function goBack() {
    play('back')
    back()
  }
  const padLabel = $derived(
    app.pads.length === 0 ? 'Nenhum controle conectado' : `${app.pads.length} controle(s): ${app.pads.map((p) => p.name).join(', ')}`,
  )
</script>

<header class="top">
  {#if canBack}
    <!-- voltar sempre no mesmo lugar, em toda tela que veio de outra -->
    <button class="back-btn" data-nav onclick={goBack}>
      <Icon name={toHome ? 'home' : 'back'} size="1.2em" />
      <span class="sub">{toHome ? 'Início' : 'Voltar'}</span>
      <span aria-hidden="true"><Glyph button="east" family={glyphFamily()} /></span>
    </button>
  {/if}
  <div class="brand pixel" aria-hidden="true">ARK<span>FLIX</span></div>
  <div class="middle">{@render children?.()}</div>
  <div class="right">
    {#if app.update?.available && !isKids()}
      <!-- aviso de versão nova: leva pra aba Atualizações -->
      <button class="upd-btn" data-nav title="Atualização disponível" onclick={() => go({ name: 'settings', tab: 'updates' })}>
        <Icon name="download" size="1.2em" /> <span>Atualizar</span> <span class="ver">{versionArrow(app.update)}</span>
      </button>
    {/if}
    {#if actions}
      <!-- como o canto do PS5: busca e avisos em blocos iguais, perfil com nome -->
      <button class="icon-btn" data-nav aria-label="Buscar jogos" title="Buscar" onclick={() => go({ name: 'search' })}>
        <Icon name="search" size="1.35em" />
      </button>
      <button class="icon-btn" data-nav aria-label={news ? `Central: ${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : 'Central de downloads e avisos'} title="Central" onclick={() => go({ name: 'activity' })}>
        <Icon name="bell" size="1.35em" />
        {#if news}<span class="dot" aria-hidden="true"></span>{/if}
      </button>
      {#if me}
        <!-- o perfil abre "Você": perfil, configurações, estatísticas, energia -->
        <button class="profile-btn" data-nav aria-label={`Você: ${me.name} (perfil e configurações)`} style:--profile={me.color} onclick={() => go({ name: 'you' })}>
          <Avatar avatar={me.avatar} color={me.color} size="2.4rem" />
          <span class="who sub">{me.name}</span>
        </button>
      {/if}
      <span class="sep" aria-hidden="true"></span>
    {/if}
    <span class="status" class:on={app.pads.length > 0} role="img" aria-label={padLabel} title={padLabel}>
      <Icon name="pad" size="1.3em" />
      {#if app.pads.length}<span class="count hud">{app.pads.length}</span>{/if}
    </span>
    {#if app.mode === 'desktop'}
      <!-- energia sempre à mão: minimizar, tela cheia, fechar -->
      <button class="icon-btn power" data-nav aria-label="Energia" title="Energia" onclick={() => (app.power = true)}>
        <Icon name="power" size="1.35em" />
      </button>
    {/if}
    <span class="when">
      <time class="clock sub" datetime={now.toISOString()}>{clock}</time>
      <span class="date">{date}</span>
    </span>
  </div>
</header>

<style>
  /* fonte de texto: versão é número pequeno, em pixel não dava pra ler */
  .upd-btn {
    display: inline-flex;
    gap: 0.6rem;
    align-items: center;
    height: 3rem;
    padding: 0 1rem;
    font: inherit;
    font-weight: 700;
    font-size: 1rem;
    color: #000;
    background: var(--focus);
    border: 3px solid var(--focus);
    box-shadow: 0 0 12px var(--focus);
    cursor: pointer;
    animation: upd-pulse 2s steps(2) infinite;
  }
  .upd-btn .ver {
    font-weight: 400;
  }
  .upd-btn:focus-visible,
  .upd-btn:hover {
    border-color: #fff;
  }
  @keyframes upd-pulse {
    50% {
      box-shadow: 0 0 2px var(--focus);
    }
  }
  :global(.reduce-motion) .upd-btn {
    animation: none;
  }
  .dot {
    position: absolute;
    top: 0.25rem;
    right: 0.25rem;
    width: 0.6rem;
    height: 0.6rem;
    background: var(--focus);
  }
  /* blocos iguais, como as caixas dos consoles */
  .icon-btn {
    position: relative;
    width: 3rem;
    height: 3rem;
    display: grid;
    place-items: center;
    color: var(--text);
    background: color-mix(in srgb, var(--surface) 70%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
    cursor: pointer;
  }
  .icon-btn:hover,
  .profile-btn:hover {
    border-color: var(--accent);
  }
  .profile-btn {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 0.7rem;
    height: 3rem;
    padding: 0 1rem 0 0.25rem;
    font: inherit;
    color: #fff;
    background: color-mix(in srgb, var(--profile) 22%, var(--surface));
    border: 2px solid color-mix(in srgb, var(--profile) 70%, transparent);
    cursor: pointer;
  }
  .who {
    font-size: 1.05rem;
    max-width: 9rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .power {
    color: #ff6b81;
  }
  .sep {
    width: 2px;
    height: 2rem;
    background: color-mix(in srgb, var(--text) 18%, transparent);
  }
  .top {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 1rem 2.5rem;
    min-height: 4.5rem;
  }
  .brand {
    font-family: var(--font-title);
    font-size: 1.9rem;
    font-size-adjust: 0.42;
    letter-spacing: 0.08em;
    color: #fff;
    text-shadow: 0 0 6px var(--accent), 0 0 16px var(--accent), 3px 3px 0 #000;
  }
  .brand span {
    color: var(--accent);
  }
  .middle {
    flex: 1;
    min-width: 0;
  }
  .right {
    display: flex;
    align-items: center;
    gap: 0.8rem;
  }
  .status {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    color: color-mix(in srgb, var(--text) 45%, transparent);
  }
  /* controle conectado acende */
  .status.on {
    color: #5dc44f;
  }
  .count {
    font-size: 1.2rem;
  }
  .when {
    display: grid;
    justify-items: end;
    line-height: 1;
    gap: 0.25rem;
  }
  .clock {
    font-size: 1.55rem;
    color: #fff;
  }
  .date {
    font-size: 0.8rem;
    color: var(--muted);
  }
  .back-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    height: 3rem;
    padding: 0 0.9rem;
    font: inherit;
    font-size: 1.05rem;
    color: #fff;
    background: color-mix(in srgb, var(--surface) 70%, transparent);
    border: 2px solid color-mix(in srgb, var(--text) 18%, transparent);
    cursor: pointer;
  }
  .back-btn:hover {
    border-color: var(--accent);
  }
</style>
