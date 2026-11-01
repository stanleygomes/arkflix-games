<script lang="ts">
  // Teclado na tela, pra digitar só com o controle. O campo de texto de verdade
  // fica acima (dá pra digitar no teclado físico também).
  import { play } from '../lib/sound'

  let {
    value = $bindable(''),
    onsubmit,
    submitLabel = 'OK',
    max = 60,
    autofocus = false,
  }: { value?: string; onsubmit?: () => void; submitLabel?: string; max?: number; autofocus?: boolean } = $props()

  let upper = $state(false)
  const rows = ['1234567890', 'qwertyuiop', 'asdfghjkl-', "zxcvbnm.'&"]

  function type(ch: string) {
    if (value.length >= max) return play('error')
    value += upper ? ch.toUpperCase() : ch
    play('move')
  }
</script>

<div class="osk" role="group" aria-label="Teclado na tela">
  {#each rows as row, r (r)}
    <div class="row">
      {#each row.split('') as ch, c (c)}
        <button
          type="button"
          class="key pixel"
          data-nav
          data-key={ch}
          data-sfx="none"
          data-nav-default={autofocus && r === 1 && c === 0 ? '' : undefined}
          onclick={() => type(ch)}>{upper ? ch.toUpperCase() : ch}</button
        >
      {/each}
    </div>
  {/each}
  <div class="row">
    <button type="button" class="key wide" data-nav data-sfx="none" aria-pressed={upper} onclick={() => (upper = !upper)}>Maiúsculas</button>
    <button type="button" class="key space" data-nav data-sfx="none" onclick={() => type(' ')}>Espaço</button>
    <button type="button" class="key wide" data-nav data-sfx="none" onclick={() => ((value = value.slice(0, -1)), play('back'))}>Apagar</button>
    <button type="button" class="key wide" data-nav data-sfx="none" onclick={() => ((value = ''), play('back'))}>Limpar</button>
    {#if onsubmit}
      <button type="button" class="key wide ok" data-nav onclick={onsubmit}>{submitLabel}</button>
    {/if}
  </div>
</div>

<style>
  .osk {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    padding: 0.8rem;
    background: color-mix(in srgb, var(--bg) 60%, #000);
    border: 3px solid color-mix(in srgb, var(--text) 25%, transparent);
    width: fit-content;
  }
  .row {
    display: flex;
    gap: 0.4rem;
  }
  .key {
    font: inherit;
    font-family: var(--font-title);
    min-width: 2.6rem;
    height: 2.6rem;
    font-size: 1.05rem;
    color: var(--text);
    background: var(--surface);
    border: 2px solid color-mix(in srgb, var(--text) 30%, var(--surface));
    box-shadow: 2px 2px 0 var(--shadow);
    cursor: pointer;
  }
  .key.wide {
    font-family: var(--font-body);
    font-weight: 700;
    font-size: 0.85rem;
    padding: 0 0.6rem;
  }
  .key.space {
    flex: 1;
    font-family: var(--font-body);
    font-weight: 700;
    font-size: 0.85rem;
  }
  .key.ok {
    background: var(--accent);
    color: var(--on-accent);
  }
  .key[aria-pressed='true'] {
    border-color: var(--accent);
  }
</style>
