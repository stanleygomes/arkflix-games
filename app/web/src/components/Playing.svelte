<script lang="ts">
  // Jogo abrindo/aberto: "PLAYER 1 START" piscando sobre a capa enquanto o
  // emulador sobe (no desktop a janela some logo depois; no navegador fica).
  import { panelIn } from '../lib/motion'
  import { app, game, glyphFamily } from '../lib/state.svelte'
  import Glyph from './Glyph.svelte'

  const p = $derived(app.playing)
  const family = $derived(glyphFamily())
  const cover = $derived(p ? game(p.id)?.cover : '')
</script>

{#if p?.state === 'running'}
  <div class="playing" role="dialog" aria-modal="true" aria-labelledby="playing-title">
    {#if cover}<div class="cover" style:background-image={`url("${cover}")`} aria-hidden="true"></div>{/if}
    <div class="box" in:panelIn>
      <p class="start pixel" aria-hidden="true">PLAYER 1 START</p>
      <h2 id="playing-title" class="pixel">Jogando {p.name}</h2>
      <p class="muted">Feche o jogo pra voltar. Atalhos dentro do jogo:</p>
      <ul class="tips">
        <li><Glyph button="select" {family} /> + <Glyph button="north" {family} /> menu do emulador · Esc</li>
        <li><Glyph button="select" {family} /> + <Glyph button="start" {family} /> sair do jogo · F10</li>
        <li><Glyph button="select" {family} /> + <Glyph button="r1" {family} /> salvar · <Glyph button="select" {family} /> + <Glyph button="l1" {family} /> carregar (menos PSP e PS2)</li>
      </ul>
    </div>
  </div>
{/if}

<style>
  .playing {
    position: fixed;
    inset: 0;
    z-index: 90;
    background: #000;
    display: grid;
    place-items: center;
    overflow: hidden;
  }
  .cover {
    position: absolute;
    inset: -2rem;
    background-size: cover;
    background-position: center;
    image-rendering: pixelated;
    filter: blur(6px) saturate(1.3);
    opacity: 0.28;
  }
  .start {
    font-size: clamp(1.8rem, 5vw, 3.6rem);
    color: var(--focus);
    text-shadow: 0 0 10px var(--focus), 0 0 30px var(--focus);
    letter-spacing: 0.1em;
    animation: start-blink 0.5s steps(1) 4 both;
  }
  @keyframes start-blink {
    50% {
      opacity: 0;
    }
  }
  .box {
    position: relative;
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
</style>
