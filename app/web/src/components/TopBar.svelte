<script lang="ts">
  import type { Snippet } from 'svelte'
  import { unread } from '../lib/activity.svelte'
  import { app, currentProfile, go } from '../lib/state.svelte'
  import Avatar from './Avatar.svelte'
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
  const padLabel = $derived(
    app.pads.length === 0 ? 'Nenhum controle conectado' : `${app.pads.length} controle(s): ${app.pads.map((p) => p.name).join(', ')}`,
  )
</script>

<header class="top">
  <div class="brand pixel" aria-hidden="true">ARK<span>FLIX</span></div>
  <div class="middle">{@render children?.()}</div>
  <div class="right">
    {#if actions}
      <button class="icon-btn" data-nav aria-label="Buscar jogos" onclick={() => go({ name: 'search' })}>
        <Icon name="search" />
      </button>
      {#if me}
        <!-- o avatar abre "Você": perfil, configurações, estatísticas, energia -->
        <button class="profile-btn" data-nav aria-label={`Você: ${me.name} (perfil e configurações)${news ? `, ${news} aviso${news === 1 ? '' : 's'} novo${news === 1 ? '' : 's'}` : ''}`} onclick={() => go({ name: 'you' })}>
          <Avatar avatar={me.avatar} color={me.color} size="2.1rem" />
          {#if news}<span class="dot" aria-hidden="true"></span>{/if}
        </button>
      {/if}
    {/if}
    <span class="status" role="img" aria-label={padLabel} title={padLabel}>
      <Icon name="pad" />
      <span class="count">{app.pads.length}</span>
    </span>
    <time class="clock pixel" datetime={now.toISOString()}>{clock}</time>
  </div>
</header>

<style>
  .dot {
    position: absolute;
    top: -0.35rem;
    right: -0.35rem;
    width: 0.8rem;
    height: 0.8rem;
    background: var(--focus);
    box-shadow: 0 0 8px var(--focus);
  }
  .profile-btn {
    position: relative;
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
