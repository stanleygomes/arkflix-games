<script lang="ts">
  // Protetor de tela: liga depois de X minutos parado (nunca com jogo aberto).
  // "Cobrinha": jogo da cobrinha jogando sozinho em fósforo verde. "Vitrine":
  // capas da coleção em estilo attract mode de fliperama. Qualquer entrada sai.
  import { idle, idleFor, sleep, watchPointer } from '../lib/idle.svelte'
  import { choose, newGame, rng, step, stuck, type Snake } from '../lib/snake'
  import { app, system } from '../lib/state.svelte'
  import type { Game } from '../lib/types'

  const W = 32
  const H = 18
  const TICK = 85

  const cfg = $derived(app.settings?.screensaver)
  const reduce = $derived(
    !!app.settings?.a11y.reduceMotion ||
      (typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches),
  )

  // relógio da inatividade
  $effect(() => {
    watchPointer()
    const iv = setInterval(() => {
      if (!cfg?.enabled || idle.active || app.playing || !app.ready) return
      if (idleFor() >= cfg.minutes * 60_000) sleep()
    }, 1000)
    return () => clearInterval(iv)
  })

  // com jogo aberto o protetor sai
  $effect(() => {
    if (app.playing) idle.active = false
  })

  // ---- cobrinha
  let canvas = $state<HTMLCanvasElement | null>(null)
  let snake = $state<Snake | null>(null)
  let hi = $state(0)

  function draw(s: Snake, blink: boolean) {
    const ctx = canvas?.getContext('2d')
    if (!ctx || !canvas) return
    const cell = canvas.width / W
    ctx.fillStyle = '#020a04'
    ctx.fillRect(0, 0, canvas.width, canvas.height)
    // grade bem fraquinha, como fósforo queimado
    ctx.fillStyle = 'rgba(51, 255, 102, 0.05)'
    for (let i = 0; i < W * H; i++) ctx.fillRect((i % W) * cell + cell / 2 - 1, Math.floor(i / W) * cell + cell / 2 - 1, 2, 2)
    ctx.shadowColor = '#33ff66'
    ctx.shadowBlur = cell * 0.6
    s.body.forEach((c, i) => {
      ctx.fillStyle = i === 0 ? '#b6ffc8' : `rgba(51, 255, 102, ${Math.max(0.45, 1 - i / (s.body.length * 1.4))})`
      ctx.fillRect((c % W) * cell + 1, Math.floor(c / W) * cell + 1, cell - 2, cell - 2)
    })
    if (s.food >= 0 && blink) {
      ctx.shadowColor = '#ffb000'
      ctx.fillStyle = '#ffb000'
      const x = (s.food % W) * cell
      const y = Math.floor(s.food / W) * cell
      ctx.fillRect(x + cell * 0.2, y + cell * 0.2, cell * 0.6, cell * 0.6)
    }
    ctx.shadowBlur = 0
  }

  $effect(() => {
    if (!idle.active || cfg?.mode !== 'snake' || !canvas) return
    const rnd = rng(Date.now())
    let s = newGame(W, H, rnd)
    if (reduce) {
      // versão parada: um quadro de uma partida já em andamento
      for (let i = 0; i < 400 && s.alive; i++) s = step(s, rnd)
      snake = s
      draw(s, true)
      return
    }
    let t = 0
    let dead = 0
    const iv = setInterval(() => {
      t++
      if (!s.alive || stuck(s) || choose(s) === null) {
        if (++dead > 18) {
          s = newGame(W, H, rnd)
          dead = 0
        }
      } else {
        s = step(s, rnd)
        hi = Math.max(hi, s.score)
      }
      snake = s
      draw(s, t % 6 < 4)
    }, TICK)
    return () => clearInterval(iv)
  })

  // ---- vitrine
  const covers = $derived(app.games.filter((g) => g.cover))
  let shown = $state<Game | null>(null)
  $effect(() => {
    if (!idle.active || cfg?.mode !== 'showcase') return
    const pool = [...covers]
    if (!pool.length) return
    const pick = () => (shown = pool[Math.floor(Math.random() * pool.length)])
    pick()
    if (reduce) return
    const iv = setInterval(pick, 5000)
    return () => clearInterval(iv)
  })
</script>

{#if idle.active && cfg}
  <div class="saver" class:reduce role="dialog" aria-modal="true" aria-label="Protetor de tela" data-mode={cfg.mode}>
    {#if cfg.mode === 'snake'}
      <div class="crt">
        <p class="score pixel" aria-hidden="true">
          <span>SCORE {String(snake?.score ?? 0).padStart(4, '0')}</span>
          <span>HI {String(hi).padStart(4, '0')}</span>
        </p>
        <canvas bind:this={canvas} width={W * 24} height={H * 24} aria-hidden="true"></canvas>
      </div>
    {:else if shown}
      {@const sys = system(shown.system)}
      <div class="showcase">
        {#key shown.id}
          <img src={shown.cover} alt="" />
        {/key}
        <p class="name pixel">{shown.name}</p>
        <p class="sys">{sys?.name ?? shown.system}</p>
      </div>
    {:else}
      <p class="name pixel">ARKFLIX</p>
    {/if}
    <p class="coin pixel">{cfg.mode === 'snake' ? 'PRESS ANY BUTTON' : 'INSERT COIN'}</p>
    <p class="sr-only">Protetor de tela. Aperte qualquer botão ou tecla para voltar.</p>
  </div>
{/if}

<style>
  .saver {
    position: fixed;
    inset: 0;
    z-index: 10000; /* acima das camadas de tubo (o protetor tem as próprias) */
    background: #000;
    color: #33ff66;
    display: grid;
    place-content: center;
    justify-items: center;
    gap: 1.2rem;
    cursor: none;
  }
  .saver[data-mode='showcase'] {
    color: #fff;
  }
  .crt {
    display: grid;
    gap: 0.6rem;
    padding: 1rem;
    border-radius: 18px;
    box-shadow:
      0 0 0 4px #0a2a12,
      0 0 60px rgba(51, 255, 102, 0.18);
  }
  .score {
    display: flex;
    justify-content: space-between;
    font-size: 1.1rem;
    text-shadow: 0 0 8px #33ff66;
  }
  canvas {
    width: min(86vw, 150vh);
    max-height: 72vh;
    aspect-ratio: 32 / 18;
    image-rendering: pixelated;
    border-radius: 10px;
  }
  .showcase {
    display: grid;
    justify-items: center;
    gap: 0.8rem;
  }
  .showcase img {
    height: 58vh;
    max-width: 80vw;
    object-fit: contain;
    image-rendering: pixelated;
    box-shadow: 10px 10px 0 #222;
    animation: slide 0.5s steps(5);
  }
  .name {
    font-size: 1.6rem;
    text-align: center;
  }
  .sys {
    color: #bbb;
  }
  .coin {
    color: #ffd23f;
    font-size: 1.2rem;
    letter-spacing: 0.15em;
    animation: blink 1.1s steps(1) infinite; /* liga/desliga, sem meio-termo */
  }
  .reduce .coin,
  .reduce img {
    animation: none;
  }
  @keyframes slide {
    from {
      transform: translateX(12%);
      opacity: 0;
    }
  }
  @keyframes blink {
    50% {
      opacity: 0;
    }
  }
</style>
