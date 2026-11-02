<script lang="ts">
  // "Atualizando": cobre tudo enquanto o app instala e reabre.
  import { app } from '../lib/state.svelte'
</script>

{#if app.restarting}
  <div class="restarting" role="alertdialog" aria-modal="true" aria-labelledby="restart-title" aria-describedby="restart-text">
    <p class="logo pixel" aria-hidden="true">ARK<span>FLIX</span></p>
    <h2 id="restart-title" class="pixel">Atualizando pra {app.restarting}</h2>
    <p id="restart-text" class="muted">O app fecha e abre de novo sozinho em alguns segundos.</p>
    <div class="dots pixel" aria-hidden="true"><span>.</span><span>.</span><span>.</span></div>
  </div>
{/if}

<style>
  .restarting {
    position: fixed;
    inset: 0;
    z-index: 30000;
    background: #000;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 1rem;
    text-align: center;
  }
  .logo {
    font-size: clamp(2.4rem, 7vw, 5rem);
    color: #fff;
    text-shadow: 0 0 12px var(--accent), 0 0 40px var(--accent);
  }
  .logo span {
    color: var(--accent);
  }
  h2 {
    font-size: 1.4rem;
    color: var(--accent-2);
  }
  .dots span {
    font-size: 2rem;
    animation: dot 1.2s steps(1) infinite;
  }
  .dots span:nth-child(2) {
    animation-delay: 0.4s;
  }
  .dots span:nth-child(3) {
    animation-delay: 0.8s;
  }
  @keyframes dot {
    50% {
      opacity: 0;
    }
  }
  :global(.reduce-motion) .dots span {
    animation: none;
  }
</style>
