<script lang="ts">
  // Chão de fase synthwave atrás das telas de jogo: céu roxo→laranja pontilhado,
  // sol listrado no horizonte e grade neon na cor do console correndo pra
  // frente. Canvas 160×100 ampliado em pixel, 12 quadros por segundo, só
  // redesenha quando o degrau muda. Parado com "reduzir animações".
  import { onMount } from 'svelte'

  const W = 160
  const H = 100
  const HORIZON = 56
  const SUN_X = 104
  const SUN_R = 21
  // céu em 5 faixas com pontilhado entre elas (16-bit, sem degradê liso)
  const SKY = ['#0b0420', '#180834', '#2c0c48', '#4a1158', '#7a1b5e']
  const SUN = ['#ffec27', '#ffc93c', '#ffa300', '#ff77a8', '#ff3c7a', '#ff004d']
  const BAYER = [
    [0, 8, 2, 10],
    [12, 4, 14, 6],
    [3, 11, 1, 9],
    [15, 7, 13, 5],
  ]

  let canvas: HTMLCanvasElement

  function paintSky(ctx: CanvasRenderingContext2D) {
    for (let y = 0; y < HORIZON; y++) {
      const t = (y / HORIZON) * (SKY.length - 1)
      const band = Math.floor(t)
      const frac = t - band
      for (let x = 0; x < W; x++) {
        const next = frac * 16 > BAYER[y % 4][x % 4] && band + 1 < SKY.length
        ctx.fillStyle = SKY[next ? band + 1 : band]
        ctx.fillRect(x, y, 1, 1)
      }
    }
    // sol: metade de cima de um círculo, cortado por faixas que engrossam
    for (let y = HORIZON - SUN_R; y < HORIZON; y++) {
      const dy = HORIZON - y
      const k = (y - (HORIZON - SUN_R)) / SUN_R // 0 no topo, 1 no horizonte
      const gap = k > 0.45 && y % 4 < Math.round(k * 3)
      if (gap) continue
      const half = Math.round(Math.sqrt(SUN_R * SUN_R - dy * dy))
      ctx.fillStyle = SUN[Math.min(SUN.length - 1, Math.floor(k * SUN.length))]
      ctx.fillRect(SUN_X - half, y, half * 2, 1)
    }
  }

  function line(ctx: CanvasRenderingContext2D, x0: number, y0: number, x1: number, y1: number) {
    // Bresenham: pixel inteiro, sem serrilhado borrado
    x0 = Math.round(x0)
    y0 = Math.round(y0)
    x1 = Math.round(x1)
    y1 = Math.round(y1)
    const dx = Math.abs(x1 - x0)
    const dy = -Math.abs(y1 - y0)
    const sx = x0 < x1 ? 1 : -1
    const sy = y0 < y1 ? 1 : -1
    let err = dx + dy
    for (;;) {
      if (x0 >= 0 && x0 < W && y0 >= 0 && y0 < H) ctx.fillRect(x0, y0, 1, 1)
      if (x0 === x1 && y0 === y1) break
      const e2 = 2 * err
      if (e2 >= dy) {
        err += dy
        x0 += sx
      }
      if (e2 <= dx) {
        err += dx
        y0 += sy
      }
    }
  }

  function paintFloor(ctx: CanvasRenderingContext2D, phase: number, color: string) {
    ctx.fillStyle = '#0b0420'
    ctx.fillRect(0, HORIZON, W, H - HORIZON)
    const cx = W / 2
    // brilho: a mesma grade, deslocada, em tom escuro (sem blur)
    for (const [dx, dy, c] of [
      [1, 0, glow(color)],
      [0, 1, glow(color)],
      [0, 0, color],
    ] as const) {
      ctx.fillStyle = c
      for (let i = -12; i <= 12; i++) line(ctx, cx + i * 5 + dx, HORIZON + dy, cx + i * 30 + dx, H + dy)
      // linhas que correm pra frente; perto do horizonte elas se juntam, então
      // as que ficariam a menos de 3 px da anterior não são desenhadas
      let last = -99
      for (let k = 9; k >= 0; k--) {
        const depth = k + 1 - phase
        if (depth <= 0.35) continue
        const y = Math.round(HORIZON + (H - HORIZON) * (0.9 / depth))
        if (y >= H || y - last < 3) continue
        last = y
        ctx.fillRect(0, y + dy, W, 1)
      }
    }
    ctx.fillStyle = '#ffa300' // horizonte âmbar
    ctx.fillRect(0, HORIZON, W, 1)
  }

  // cor do brilho: o acento misturado com o fundo do chão (o canvas não lê
  // color-mix, então a conta é feita aqui)
  function glow(hex: string): string {
    const n = Number.parseInt(hex.slice(1), 16)
    const base = [0x0b, 0x04, 0x20]
    const mix = [(n >> 16) & 255, (n >> 8) & 255, n & 255].map((c, i) => Math.round(c * 0.45 + base[i] * 0.55))
    return `rgb(${mix.join(' ')})`
  }

  // normaliza qualquer cor CSS simples (rgb(), #abc) pra #rrggbb
  function toHex(ctx: CanvasRenderingContext2D, color: string): string {
    ctx.fillStyle = '#ff3cac'
    ctx.fillStyle = color
    return String(ctx.fillStyle)
  }

  onMount(() => {
    const ctx = canvas.getContext('2d')
    if (!ctx) return
    const sky = document.createElement('canvas')
    sky.width = W
    sky.height = HORIZON
    const skyCtx = sky.getContext('2d')
    if (skyCtx) paintSky(skyCtx)
    const still = () =>
      document.body.classList.contains('reduce-motion') || matchMedia('(prefers-reduced-motion: reduce)').matches
    const STEPS = 12 // 12 degraus por "casa" da grade = 1 s a 12 fps
    let step = -1
    const render = () => {
      const s = still() ? 0 : (step + 1) % STEPS
      if (s === step) return // parado: não redesenha
      step = s
      const raw = getComputedStyle(document.documentElement).getPropertyValue('--accent').trim()
      const accent = toHex(ctx, raw || '#ff3cac')
      ctx.drawImage(sky, 0, 0)
      paintFloor(ctx, s / STEPS, accent)
    }
    render()
    // timer de 12 fps (não requestAnimationFrame): só trabalha no degrau
    const id = window.setInterval(() => {
      if (!document.hidden) render()
    }, 1000 / 12)
    return () => clearInterval(id)
  })
</script>

<canvas class="synthwave" bind:this={canvas} width={W} height={H} aria-hidden="true"></canvas>

<style>
  .synthwave {
    position: fixed;
    inset: 0;
    width: 100%;
    height: 100%;
    z-index: 0;
    image-rendering: pixelated;
    pointer-events: none;
    object-fit: cover;
    opacity: 0.7; /* cenário, não protagonista: o texto por cima manda */
  }
  :global(body.high-contrast) .synthwave {
    display: none;
  }
</style>
