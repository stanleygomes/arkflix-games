<script lang="ts">
  // Tela de ligar (só no app desktop): logo acendendo em néon com o som de
  // boot, ~1,6 s. Qualquer botão pula. Com "reduzir animações", não aparece.
  import { onMount } from 'svelte'
  import { play, unlock } from '../lib/sound'
  import { app } from '../lib/state.svelte'

  let show = $state(false)

  onMount(() => {
    const reduce = app.settings?.a11y.reduceMotion || matchMedia('(prefers-reduced-motion: reduce)').matches
    if (app.mode !== 'desktop' || reduce) return
    show = true
    unlock()
    play('boot')
    const t = setTimeout(() => (show = false), 1700)
    const skip = () => (show = false)
    window.addEventListener('keydown', skip, { once: true, capture: true })
    window.addEventListener('pointerdown', skip, { once: true, capture: true })
    return () => {
      clearTimeout(t)
      window.removeEventListener('keydown', skip, { capture: true })
      window.removeEventListener('pointerdown', skip, { capture: true })
    }
  })
</script>

{#if show}
  <div class="boot" aria-hidden="true">
    <!-- TV ligando: uma linha no meio abre até a tela cheia, com um clarão -->
    <div class="tube">
      <p class="logo pixel">ARK<span>FLIX</span></p>
      <p class="sub pixel">GAMES</p>
      <div class="scan"></div>
    </div>
  </div>
{/if}

<style>
  .boot {
    position: fixed;
    inset: 0;
    z-index: 20000;
    background: #000;
    animation: boot-out 1.7s steps(1) forwards;
  }
  .tube {
    position: absolute;
    inset: 0;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 0.8rem;
    background: radial-gradient(ellipse at center, #0c0608, #000 75%);
    animation: power-on 0.45s steps(5) both;
  }
  @keyframes power-on {
    0% {
      transform: scale(0.6, 0.004);
      filter: brightness(4);
    }
    40% {
      transform: scale(1, 0.004);
      filter: brightness(3);
    }
    80% {
      transform: scale(1, 1);
      filter: brightness(1.8);
    }
    100% {
      transform: none;
      filter: none;
    }
  }
  .logo {
    font-size: clamp(3rem, 9vw, 7rem);
    color: #fff;
    letter-spacing: 0.08em;
    text-shadow: 0 0 12px var(--accent), 0 0 40px var(--accent);
    animation: neon 0.9s steps(6) both;
  }
  .logo span {
    color: var(--accent);
  }
  .sub {
    color: var(--accent-2);
    letter-spacing: 0.6em;
    text-shadow: 0 0 10px var(--accent-2);
    animation: rise 0.5s 0.6s both;
  }
  .scan {
    position: absolute;
    inset: 0;
    background: linear-gradient(transparent 0 48%, color-mix(in srgb, var(--accent-2) 35%, transparent) 50%, transparent 52%);
    animation: sweep 1.1s ease-out both;
  }
  @keyframes neon {
    0%,
    20%,
    40% {
      opacity: 0;
    }
    30%,
    60%,
    100% {
      opacity: 1;
    }
  }
  @keyframes rise {
    from {
      opacity: 0;
      transform: translateY(0.6rem);
    }
  }
  @keyframes sweep {
    from {
      transform: translateY(-60%);
    }
    to {
      transform: translateY(60%);
      opacity: 0;
    }
  }
  @keyframes boot-out {
    94% {
      opacity: 1;
    }
    100% {
      opacity: 0;
    }
  }
</style>
