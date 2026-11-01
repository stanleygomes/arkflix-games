<script lang="ts">
  import { tick } from 'svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import Icon, { type IconName } from '../components/Icon.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { pushHandler } from '../lib/input'
  import { current, focusEl } from '../lib/nav'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { play } from '../lib/sound'
  import { app, type SettingsTab } from '../lib/state.svelte'
  import A11yPanel from './settings/A11yPanel.svelte'
  import AboutPanel from './settings/AboutPanel.svelte'
  import ControlsPanel from './settings/ControlsPanel.svelte'
  import LibraryPanel from './settings/LibraryPanel.svelte'

  let { tab = 'library' }: { tab?: SettingsTab } = $props()
  let active = $state<SettingsTab>('library')
  $effect.pre(() => {
    active = tab
  })

  const tabs: { id: SettingsTab; label: string; icon: IconName }[] = [
    { id: 'library', label: 'Biblioteca', icon: 'folder' },
    { id: 'controls', label: 'Controles', icon: 'pad' },
    { id: 'a11y', label: 'Acessibilidade', icon: 'sliders' },
    { id: 'about', label: 'Sobre', icon: 'info' },
  ]

  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
  })

  $effect(() => {
    tick().then(() => focusEl(document.querySelector<HTMLElement>(`[data-settings-tab="${active}"]`)))
  })

  function cycle(delta: number) {
    const i = tabs.findIndex((t) => t.id === active)
    active = tabs[(i + delta + tabs.length) % tabs.length].id
    play('move')
  }

  $effect(() =>
    pushHandler((a) => {
      // painel em foco: cima/baixo rolam o texto; na borda, voltam a navegar
      const el = current()
      if (el?.hasAttribute('data-panel') && (a === 'up' || a === 'down')) {
        const before = el.scrollTop
        el.scrollBy({ top: a === 'down' ? 120 : -120 })
        if (el.scrollTop !== before) return true
      }
      if (a === 'prevTab') {
        cycle(-1)
        return true
      }
      if (a === 'nextTab') {
        cycle(1)
        return true
      }
      return a === 'details' || a === 'menu'
    }),
  )
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="layout">
    <nav class="tabs" aria-label="Seções das configurações">
      <h1 class="pixel">Configurações</h1>
      {#each tabs as t (t.id)}
        <button
          class="tab cursor"
          data-nav
          data-settings-tab={t.id}
          aria-current={active === t.id ? 'page' : undefined}
          onfocus={() => (active = t.id)}
          onclick={() => (active = t.id)}
        >
          <Icon name={t.icon} /> {t.label}
        </button>
      {/each}
    </nav>
    <!-- focável: dá pra rolar o painel com teclado/controle mesmo sem botão dentro -->
    <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
    <section class="panel content" tabindex="0" data-nav={active === 'about' ? '' : undefined} data-panel aria-label={tabs.find((t) => t.id === active)?.label}>
      {#if active === 'library'}
        <LibraryPanel />
      {:else if active === 'controls'}
        <ControlsPanel />
      {:else if active === 'a11y'}
        <A11yPanel />
      {:else}
        <AboutPanel />
      {/if}
    </section>
  </main>
  <HelpBar
    hints={[
      { button: 'south', label: 'Escolher' },
      { button: 'l1', label: 'Seção anterior' },
      { button: 'r1', label: 'Próxima seção' },
      { button: 'east', label: 'Voltar' },
    ]}
  />
</div>

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .layout {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: 16rem 1fr;
    gap: 2rem;
    padding: 0.5rem 2.5rem 1rem 3.5rem;
  }
  .tabs {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  h1 {
    font-size: 1.8rem;
    margin-bottom: 1rem;
  }
  .tab {
    font: inherit;
    font-weight: 700;
    font-size: 1.05rem;
    text-align: left;
    color: var(--muted);
    background: transparent;
    border: 3px solid transparent;
    padding: 0.7rem 0.9rem;
    display: flex;
    gap: 0.6rem;
    align-items: center;
    cursor: pointer;
  }
  .tab[aria-current='page'] {
    color: var(--text);
    background: var(--surface);
    border-color: color-mix(in srgb, var(--text) 25%, var(--surface));
  }
  .content {
    overflow-y: auto;
    outline-offset: -4px;
    min-height: 0;
  }
</style>
