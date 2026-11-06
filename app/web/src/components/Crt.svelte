<script lang="ts">
  // Tubo de TV por cima do app inteiro: moldura arredondada, cantos escuros,
  // bojo do vidro, reflexo, linhas e grão. Só decoração: sem clique, fora da
  // árvore de acessibilidade. As linhas somem no centro (onde fica o texto) e
  // engrossam na borda. Alto contraste desliga tudo; "reduzir animações"
  // congela o grão e tira a cintilação.
  // paused: protetor de tela ligado, o grão sai (nada animando por trás)
  let { paused = false }: { paused?: boolean } = $props()

  let canvas = $state<HTMLCanvasElement | null>(null)

  $effect(() => {
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const { width: w, height: h } = canvas
    const img = ctx.createImageData(w, h)
    const draw = () => {
      const d = img.data
      for (let i = 0; i < d.length; i += 4) {
        const v = (Math.random() * 255) | 0
        d[i] = d[i + 1] = d[i + 2] = v
        d[i + 3] = 255
      }
      ctx.putImageData(img, 0, 0)
    }
    draw()
    const still = () =>
      document.body.classList.contains('reduce-motion') || matchMedia('(prefers-reduced-motion: reduce)').matches
    // 12 quadros por segundo: grão de sinal, não de vídeo moderno
    const id = window.setInterval(() => {
      if (!still() && !document.hidden) draw()
    }, 1000 / 12)
    return () => clearInterval(id)
  })
</script>

<div class="crt" aria-hidden="true">
  {#if !paused}<canvas class="grain" bind:this={canvas} width="480" height="270"></canvas>{/if}
  <div class="lines"></div>
  <div class="tracking"></div>
  <div class="hum"></div>
  <div class="roll"></div>
  <div class="bulge"></div>
  <div class="glass"></div>
  <div class="flicker"></div>
  <div class="bezel"></div>
</div>
<!-- separação de cor do tubo: vermelho pra direita, azul pra esquerda (usado
     só nos títulos grandes, nunca no texto corrido) -->
<svg class="defs" aria-hidden="true" width="0" height="0">
  <filter id="crt-chroma" x="-5%" y="-5%" width="110%" height="110%" color-interpolation-filters="sRGB">
    <feColorMatrix in="SourceGraphic" type="matrix" values="1 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 1 0" result="r" />
    <feOffset in="r" dx="0.8" result="r2" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 1 0 0 0  0 0 0 0 0  0 0 0 1 0" result="g" />
    <feColorMatrix in="SourceGraphic" type="matrix" values="0 0 0 0 0  0 0 0 0 0  0 0 1 0 0  0 0 0 1 0" result="b" />
    <feOffset in="b" dx="-0.8" result="b2" />
    <feBlend in="r2" in2="g" mode="screen" result="rg" />
    <feBlend in="rg" in2="b2" mode="screen" />
  </filter>
</svg>

<style>
  .crt {
    position: fixed;
    inset: 0;
    z-index: 30000; /* por cima de tudo, até do boot (não pega clique) */
    pointer-events: none;
  }
  .crt > * {
    position: absolute;
    inset: 0;
  }
  .defs {
    position: absolute;
    width: 0;
    height: 0;
  }
  .grain {
    width: 100%;
    height: 100%;
    image-rendering: pixelated;
    opacity: 0.035; /* mistura normal: blend em tela cheia custa caro na TV */
  }
  /* linhas de varredura: quase nada no meio, mais fortes na borda do tubo */
  .lines {
    background: repeating-linear-gradient(to bottom, transparent 0 2px, rgb(0 0 0 / 0.45) 3px 4px);
    mask-image: radial-gradient(ellipse 75% 70% at 50% 50%, rgb(0 0 0 / 0.22) 45%, #000 100%);
  }
  /* faixa de tracking do VHS: passa no rodapé a cada ~17 s, sem cobrir texto */
  .tracking {
    top: auto;
    height: 3.5vh;
    bottom: 1.5vh;
    opacity: 0;
    background:
      repeating-linear-gradient(90deg, rgb(255 255 255 / 0.18) 0 3px, transparent 3px 7px, rgb(255 255 255 / 0.08) 7px 9px, transparent 9px 16px),
      linear-gradient(transparent, rgb(255 255 255 / 0.06), transparent);
    animation: tracking 17s steps(1) infinite;
  }
  @keyframes tracking {
    0%,
    92% {
      opacity: 0;
      transform: translateX(0);
    }
    93% {
      opacity: 1;
      transform: translateX(-6px);
    }
    95% {
      opacity: 0.7;
      transform: translateX(4px) translateY(-0.6vh);
    }
    97% {
      opacity: 1;
      transform: translateX(-2px) translateY(-1.2vh);
    }
    99%,
    100% {
      opacity: 0;
    }
  }
  /* barra de zumbido: faixa clara larga descendo devagar (interferência de
     rede elétrica dos CRTs), quase invisível sobre o texto */
  .hum {
    bottom: auto;
    height: 18vh;
    background: linear-gradient(transparent, rgb(255 255 255 / 0.035) 45%, rgb(255 255 255 / 0.05) 50%, rgb(255 255 255 / 0.035) 55%, transparent);
    animation: hum 11s steps(66) infinite;
  }
  @keyframes hum {
    from {
      transform: translateY(-20vh);
    }
    to {
      transform: translateY(105vh);
    }
  }
  /* linha clara que rola rápido (glitch "bar", disparado pelo lib/glitch.ts) */
  .roll {
    bottom: auto;
    height: 1.2vh;
    opacity: 0;
    background: linear-gradient(transparent, rgb(255 255 255 / 0.22), transparent);
  }
  :global(html[data-glitch='bar']) .roll {
    animation: roll 0.9s steps(12) both;
  }
  @keyframes roll {
    from {
      opacity: 1;
      transform: translateY(-2vh);
    }
    to {
      opacity: 1;
      transform: translateY(102vh);
    }
  }
  /* chuvisco: o grão sobe e a imagem dá uma queda de brilho */
  :global(html[data-glitch='snow']) .grain {
    opacity: 0.22;
  }
  :global(html[data-glitch='snow']) .flicker {
    animation: none;
    opacity: 0.12;
  }

  /* bojo: centro levemente aceso, borda escura, quatro cantos fundos */
  .bulge {
    background:
      radial-gradient(circle at 0 0, rgb(0 0 0 / 0.6), transparent 16%),
      radial-gradient(circle at 100% 0, rgb(0 0 0 / 0.6), transparent 16%),
      radial-gradient(circle at 0 100%, rgb(0 0 0 / 0.6), transparent 16%),
      radial-gradient(circle at 100% 100%, rgb(0 0 0 / 0.6), transparent 16%),
      radial-gradient(ellipse 80% 75% at 50% 45%, rgb(255 255 255 / 0.03), transparent 65%, rgb(0 0 0 / 0.4) 100%);
  }
  /* reflexo no vidro, canto superior esquerdo */
  .glass {
    background: radial-gradient(ellipse 45% 30% at 22% 12%, rgb(255 255 255 / 0.06), transparent 70%);
  }
  /* cintilação rara e irregular: duas quedas curtas a cada ~7 s */
  .flicker {
    background: #000;
    opacity: 0;
    animation: flicker 7.3s steps(1) infinite;
  }
  @keyframes flicker {
    0%,
    61%,
    63%,
    88%,
    90%,
    100% {
      opacity: 0;
    }
    62% {
      opacity: 0.1;
    }
    89% {
      opacity: 0.14;
    }
  }
  /* moldura: o tubo tem cantos arredondados, o resto é o gabinete */
  .bezel {
    inset: var(--tube);
    border-radius: calc(var(--tube) * 2.4);
    box-shadow:
      0 0 0 100vmax #050407,
      inset 0 0 0 2px rgb(255 255 255 / 0.05),
      inset 0 0 18px 6px rgb(0 0 0 / 0.75);
  }
  :global(body.high-contrast) .crt {
    display: none;
  }
  :global(body.reduce-motion) :is(.flicker, .tracking, .hum, .roll) {
    display: none;
  }
  @media (prefers-reduced-motion: reduce) {
    .flicker,
    .tracking,
    .hum,
    .roll {
      display: none;
    }
  }
</style>
