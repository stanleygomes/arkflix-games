<script lang="ts">
  // Desenho do controle (Xbox, PlayStation ou estilo SNES/8BitDo). Botões
  // apertados acendem; o botão sendo configurado pisca.
  import { glyph } from '../lib/keys'
  import type { Family } from '../lib/types'

  let {
    family,
    pressed = new Set<string>(),
    highlight = '',
  }: { family: Family; pressed?: Set<string>; highlight?: string } = $props()

  const snes = $derived(family === '8bitdo' || family === 'nintendo')
  const kind = $derived(family === 'keyboard' || family === 'generic' ? 'xbox' : family)
  const name = $derived(
    { xbox: 'Xbox', playstation: 'PlayStation', '8bitdo': '8BitDo / estilo SNES', nintendo: 'Nintendo', generic: 'Genérico', keyboard: 'Teclado' }[family],
  )

  type Btn = { id: string; x: number; y: number; w?: number; h?: number; r?: number }
  const layout = $derived.by((): Btn[] => {
    if (snes) {
      return [
        { id: 'l2', x: 40, y: 18, w: 40, h: 8 }, { id: 'r2', x: 160, y: 18, w: 40, h: 8 },
        { id: 'l1', x: 34, y: 30, w: 52, h: 9 }, { id: 'r1', x: 154, y: 30, w: 52, h: 9 },
        { id: 'up', x: 54, y: 56, w: 12, h: 13 }, { id: 'down', x: 54, y: 82, w: 12, h: 13 },
        { id: 'left', x: 41, y: 69, w: 13, h: 12 }, { id: 'right', x: 66, y: 69, w: 13, h: 12 },
        { id: 'select', x: 100, y: 78, w: 16, h: 6 }, { id: 'start', x: 124, y: 78, w: 16, h: 6 },
        { id: 'home', x: 120, y: 64, r: 4 },
        { id: 'north', x: 184, y: 56, r: 8 }, { id: 'south', x: 184, y: 92, r: 8 },
        { id: 'west', x: 166, y: 74, r: 8 }, { id: 'east', x: 202, y: 74, r: 8 },
        { id: 'l3', x: 96, y: 108, r: 9 }, { id: 'r3', x: 144, y: 108, r: 9 },
      ]
    }
    const ps = kind === 'playstation'
    const dpad = ps ? { x: 60, y: 64 } : { x: 92, y: 102 }
    const stick = ps ? { x: 92, y: 102 } : { x: 60, y: 64 }
    return [
      { id: 'l2', x: 34, y: 6, w: 44, h: 10 }, { id: 'r2', x: 162, y: 6, w: 44, h: 10 },
      { id: 'l1', x: 30, y: 20, w: 52, h: 9 }, { id: 'r1', x: 158, y: 20, w: 52, h: 9 },
      { id: 'up', x: dpad.x - 6, y: dpad.y - 19, w: 12, h: 13 }, { id: 'down', x: dpad.x - 6, y: dpad.y + 6, w: 12, h: 13 },
      { id: 'left', x: dpad.x - 19, y: dpad.y - 6, w: 13, h: 12 }, { id: 'right', x: dpad.x + 6, y: dpad.y - 6, w: 13, h: 12 },
      { id: 'l3', x: stick.x, y: stick.y, r: 12 }, { id: 'r3', x: 148, y: 102, r: 12 },
      { id: 'select', x: 98, y: 56, w: 14, h: 8 }, { id: 'start', x: 128, y: 56, w: 14, h: 8 },
      { id: 'home', x: 120, y: 40, r: 7 },
      { id: 'north', x: 180, y: 48, r: 9 }, { id: 'south', x: 180, y: 80, r: 9 },
      { id: 'west', x: 164, y: 64, r: 9 }, { id: 'east', x: 196, y: 64, r: 9 },
    ]
  })
  const face = new Set(['north', 'south', 'east', 'west'])
</script>

<figure class="preview">
  <svg viewBox="0 0 240 140" role="img" aria-label={`Controle ${name}${highlight ? `, configurando ${glyph(highlight, family)}` : ''}`}>
    {#if snes}
      <path class="body" d="M40 44 h160 a36 36 0 0 1 0 72 h-160 a36 36 0 0 1 0 -72 z" />
    {:else}
      <path
        class="body"
        d="M44 30 h152 c22 0 30 18 36 46 c8 36 6 58 -12 60 c-16 2 -24 -14 -34 -28 h-92 c-10 14 -18 30 -34 28 c-18 -2 -20 -24 -12 -60 c6 -28 14 -46 36 -46 z"
      />
    {/if}
    {#each layout as b (b.id)}
      {@const cls = `btn ${pressed.has(b.id) ? 'pressed' : ''} ${highlight === b.id ? 'highlight' : ''}`}
      {#if b.r}
        <circle class={cls} data-button={b.id} cx={b.x} cy={b.y} r={b.r} />
        {#if face.has(b.id)}
          <text class="label" x={b.x} y={b.y + 3.5}>{glyph(b.id, kind === 'playstation' ? 'playstation' : snes ? '8bitdo' : 'xbox')}</text>
        {/if}
      {:else}
        <rect class={cls} data-button={b.id} x={b.x} y={b.y} width={b.w} height={b.h} />
      {/if}
    {/each}
  </svg>
  <figcaption class="muted">{name}</figcaption>
</figure>

<style>
  .preview {
    margin: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.5rem;
  }
  svg {
    width: 100%;
    max-width: 30rem;
    shape-rendering: crispEdges;
  }
  .body {
    fill: color-mix(in srgb, var(--surface) 70%, #fff 8%);
    stroke: #05040a;
    stroke-width: 3;
  }
  .btn {
    fill: #2b2a33;
    stroke: #05040a;
    stroke-width: 2;
    transition: fill 80ms;
  }
  .btn.pressed {
    fill: var(--accent);
  }
  .btn.highlight {
    stroke: var(--accent-2);
    stroke-width: 3;
    animation: flash 0.6s steps(2) infinite;
  }
  @keyframes flash {
    50% {
      stroke: #fff;
    }
  }
  .label {
    font-family: var(--font-title);
    font-size: 9px;
    fill: #f4f1e8;
    text-anchor: middle;
    pointer-events: none;
  }
</style>
