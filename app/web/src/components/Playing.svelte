<script lang="ts">
  // Tela enquanto o emulador roda (no desktop a janela some; no navegador fica aqui).
  import { app, glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  const p = $derived(app.playing)
  const family = $derived(glyphFamily())
</script>

{#if p?.state === 'running'}
  <div class="playing" role="dialog" aria-modal="true" aria-labelledby="playing-title">
    <div class="box">
      <div class="loader" aria-hidden="true"><span></span><span></span><span></span></div>
      <h2 id="playing-title" class="pixel">Jogando {p.name}</h2>
      <p class="muted">Feche o jogo pra voltar. Atalhos dentro do jogo:</p>
      <ul class="tips">
        <li><Glyph button="select" {family} /> + <Glyph button="start" {family} /> sair do jogo</li>
        <li><Glyph button="select" {family} /> + <Glyph button="r1" {family} /> salvar</li>
        <li><Glyph button="select" {family} /> + <Glyph button="l1" {family} /> carregar</li>
        <li><Glyph button="select" {family} /> + <Glyph button="north" {family} /> menu</li>
      </ul>
    </div>
  </div>
{/if}

<style>
  .playing {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: color-mix(in srgb, var(--bg) 94%, #000);
    display: grid;
    place-items: center;
  }
  .box {
    text-align: center;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    align-items: center;
  }
  h2 {
    font-size: 2.2rem;
  }
  .tips {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  .tips li {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    justify-content: center;
  }
  .loader {
    display: flex;
    gap: 0.5rem;
  }
  .loader span {
    width: 1rem;
    height: 1rem;
    background: var(--accent);
    animation: hop 0.9s steps(3) infinite;
  }
  .loader span:nth-child(2) {
    animation-delay: 0.15s;
  }
  .loader span:nth-child(3) {
    animation-delay: 0.3s;
  }
  @keyframes hop {
    50% {
      transform: translateY(-0.8rem);
    }
  }
</style>
