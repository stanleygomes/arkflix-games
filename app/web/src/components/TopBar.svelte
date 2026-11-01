<script lang="ts">
  import type { Snippet } from 'svelte'
  import { app, currentProfile, go, isKids } from '../lib/state.svelte'
  import Avatar from './Avatar.svelte'
  import Icon from './Icon.svelte'

  let { children, actions = true }: { children?: Snippet; actions?: boolean } = $props()
  let now = $state(new Date())
  $effect(() => {
    const t = setInterval(() => (now = new Date()), 15000)
    return () => clearInterval(t)
  })
  const me = $derived(currentProfile())
  const clock = $derived(now.toLocaleTimeString('pt-BR', { hour: '2-digit', minute: '2-digit' }))
  const padLabel = $derived(
    app.pads.length === 0 ? 'Nenhum controle conectado' : `${app.pads.length} controle(s): ${app.pads.map((p) => p.name).join(', ')}`,
  )
</script>

<header class="top">
  <div class="brand pixel" aria-hidden="true">ARK<span>FLIX</span></div>
  <div class="middle">{@render children?.()}</div>
  <div class="right">
    {#if actions}
      {#if me}
        <button class="profile-btn" data-nav aria-label={`Trocar perfil (${me.name})`} onclick={() => go({ name: 'profiles' })}>
          <Avatar avatar={me.avatar} color={me.color} size="2.1rem" />
        </button>
      {/if}
      <button class="icon-btn" data-nav aria-label="Buscar jogos" onclick={() => go({ name: 'search' })}>
        <Icon name="search" />
      </button>
      <button class="icon-btn" data-nav aria-label="Estatísticas" onclick={() => go({ name: 'stats' })}>
        <Icon name="chart" />
      </button>
      {#if !isKids()}
        <button class="icon-btn" data-nav aria-label="Importar jogos" onclick={() => go({ name: 'import' })}>
          <Icon name="plus" />
        </button>
        <button class="icon-btn" data-nav aria-label="Configurações" onclick={() => go({ name: 'settings' })}>
          <Icon name="gear" />
        </button>
      {/if}
    {/if}
    {#if app.mode === 'desktop'}
      <button class="icon-btn" data-nav aria-label="Energia: minimizar, tela cheia ou fechar" aria-haspopup="dialog" onclick={() => (app.power = true)}>
        <Icon name="power" />
      </button>
    {/if}
    <span class="status" role="img" aria-label={padLabel} title={padLabel}>
      <Icon name="pad" />
      <span class="count">{app.pads.length}</span>
    </span>
    <time class="clock pixel" datetime={now.toISOString()}>{clock}</time>
  </div>
</header>

<style>
  .profile-btn {
    padding: 0;
    background: none;
    border: 3px solid transparent;
    cursor: pointer;
    display: grid;
  }
  .profile-btn:focus-visible,
  .profile-btn:hover {
    border-color: var(--accent);
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
    gap: 0.9rem;
  }
  .icon-btn {
    font-size: 1.3rem;
    width: 2.75rem;
    height: 2.75rem;
    display: grid;
    place-items: center;
    color: var(--text);
    background: transparent;
    border: 3px solid transparent;
    cursor: pointer;
  }
  .icon-btn:hover {
    border-color: color-mix(in srgb, var(--text) 30%, transparent);
  }
  .status {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    font-size: 1.2rem;
    color: var(--muted);
  }
  .count {
    font-size: 0.85rem;
  }
  .clock {
    font-size: 1.2rem;
    min-width: 3.5em;
    text-align: right;
  }
</style>
