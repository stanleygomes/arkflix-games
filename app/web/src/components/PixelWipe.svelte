<script lang="ts">
  // Mosaico em blocos (transição de SNES): ao recomeçar a navegação (trocar de
  // perfil, sair da tela de entrada), a tela nova aparece bloco a bloco.
  import { reduced } from '../lib/motion'
  import { app } from '../lib/state.svelte'

  const COLS = 16
  const ROWS = 9
  let run = $state(0)
  let last = app.nav.seq
  const blocks = Array.from({ length: COLS * ROWS }, (_, i) => ({ i, delay: Math.floor(Math.random() * 8) * 45 }))

  $effect(() => {
    const { seq, dir } = app.nav
    if (seq === last) return
    last = seq
    if (dir !== 'reset' || reduced()) return
    run++
    const mine = run
    setTimeout(() => {
      if (run === mine) run = -mine // esconde depois da animação
    }, 520)
  })
</script>

{#if run > 0}
  {#key run}
    <div class="wipe" aria-hidden="true" style:--cols={COLS}>
      {#each blocks as b (b.i)}<span style:animation-delay={`${b.delay}ms`}></span>{/each}
    </div>
  {/key}
{/if}

<style>
  .wipe {
    position: fixed;
    inset: 0;
    z-index: 9000;
    display: grid;
    grid-template-columns: repeat(var(--cols), 1fr);
    pointer-events: none;
  }
  .wipe span {
    background: var(--bg);
    box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 25%, transparent);
    animation: off 0.12s steps(1) forwards;
  }
  @keyframes off {
    to {
      opacity: 0;
    }
  }
</style>
