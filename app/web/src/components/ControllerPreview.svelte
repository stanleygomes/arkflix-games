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

  type Btn = { id: string; x: number; y: number; w?: number; h?: number; r?: number; rx?: number }
  // gatilhos e bumpers vêm antes do corpo: o corpo cobre a base e eles ficam
  // "presos" no ombro, como no controle de verdade
  const shoulders = new Set(['l1', 'r1', 'l2', 'r2'])
  const layout = $derived.by((): Btn[] => {
    if (snes) {
      return [
        { id: 'l2', x: 46, y: 26, w: 40, h: 14, rx: 4 }, { id: 'r2', x: 154, y: 26, w: 40, h: 14, rx: 4 },
        { id: 'l1', x: 36, y: 34, w: 54, h: 14, rx: 5 }, { id: 'r1', x: 150, y: 34, w: 54, h: 14, rx: 5 },
        { id: 'up', x: 54, y: 62, w: 12, h: 13 }, { id: 'down', x: 54, y: 88, w: 12, h: 13 },
        { id: 'left', x: 41, y: 75, w: 13, h: 12 }, { id: 'right', x: 66, y: 75, w: 13, h: 12 },
        { id: 'select', x: 100, y: 84, w: 16, h: 6, rx: 3 }, { id: 'start', x: 124, y: 84, w: 16, h: 6, rx: 3 },
        { id: 'home', x: 120, y: 70, r: 4 },
        { id: 'north', x: 184, y: 64, r: 8 }, { id: 'south', x: 184, y: 98, r: 8 },
        { id: 'west', x: 167, y: 81, r: 8 }, { id: 'east', x: 201, y: 81, r: 8 },
        { id: 'l3', x: 96, y: 112, r: 8 }, { id: 'r3', x: 144, y: 112, r: 8 },
      ]
    }
    // PlayStation: direcional em cima, analógicos embaixo e simétricos;
    // Xbox: analógico esquerdo em cima e direcional embaixo
    const ps = kind === 'playstation'
    const dpad = ps ? { x: 62, y: 72 } : { x: 92, y: 104 }
    const lstick = ps ? { x: 92, y: 104 } : { x: 62, y: 72 }
    const face = { x: 178, y: 72 }
    return [
      { id: 'l2', x: 52, y: 20, w: 40, h: 22, rx: 6 }, { id: 'r2', x: 148, y: 20, w: 40, h: 22, rx: 6 },
      { id: 'l1', x: 44, y: 32, w: 52, h: 14, rx: 5 }, { id: 'r1', x: 144, y: 32, w: 52, h: 14, rx: 5 },
      { id: 'up', x: dpad.x - 6, y: dpad.y - 19, w: 12, h: 13 }, { id: 'down', x: dpad.x - 6, y: dpad.y + 6, w: 12, h: 13 },
      { id: 'left', x: dpad.x - 19, y: dpad.y - 6, w: 13, h: 12 }, { id: 'right', x: dpad.x + 6, y: dpad.y - 6, w: 13, h: 12 },
      { id: 'l3', x: lstick.x, y: lstick.y, r: 12 }, { id: 'r3', x: 148, y: 104, r: 12 },
      { id: 'select', x: 98, y: 66, w: 12, h: 8, rx: 3 }, { id: 'start', x: 130, y: 66, w: 12, h: 8, rx: 3 },
      { id: 'home', x: 120, y: ps ? 92 : 56, r: 6 },
      { id: 'north', x: face.x, y: face.y - 15, r: 8 }, { id: 'south', x: face.x, y: face.y + 15, r: 8 },
      { id: 'west', x: face.x - 15, y: face.y, r: 8 }, { id: 'east', x: face.x + 15, y: face.y, r: 8 },
    ]
  })
  const face = new Set(['north', 'south', 'east', 'west'])
</script>

<figure class="preview">
  <svg viewBox="0 0 240 150" role="img" aria-label={`Controle ${name}${highlight ? `, configurando ${glyph(highlight, family)}` : ''}`}>
    {#each layout.filter((b) => shoulders.has(b.id)) as b (b.id)}
      <rect class={`btn ${pressed.has(b.id) ? 'pressed' : ''} ${highlight === b.id ? 'highlight' : ''}`} data-button={b.id} x={b.x} y={b.y} width={b.w} height={b.h} rx={b.rx} />
    {/each}
    {#if snes}
      <path class="body" d="M40 46 h160 a38 38 0 0 1 0 76 h-160 a38 38 0 0 1 0 -76 z" />
    {:else}
      <path
        class="body"
        d="M62 42 C42 42 32 54 26 76 C18 106 16 132 32 138 C46 143 56 130 68 114 L172 114 C184 130 194 143 208 138 C224 132 222 106 214 76 C208 54 198 42 178 42 C150 40 90 40 62 42 Z"
      />
    {/if}
    {#each layout.filter((b) => !shoulders.has(b.id)) as b (b.id)}
      {@const cls = `btn ${pressed.has(b.id) ? 'pressed' : ''} ${highlight === b.id ? 'highlight' : ''}`}
      {#if b.r}
        <circle class={cls} data-button={b.id} cx={b.x} cy={b.y} r={b.r} />
        {#if face.has(b.id)}
          <text class="label" x={b.x} y={b.y + 3.2}>{glyph(b.id, kind === 'playstation' ? 'playstation' : snes ? '8bitdo' : 'xbox')}</text>
        {/if}
      {:else}
        <rect class={cls} data-button={b.id} x={b.x} y={b.y} width={b.w} height={b.h} rx={b.rx} />
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
    fill: color-mix(in srgb, var(--surface) 80%, #fff 6%);
    stroke: var(--accent-2);
    stroke-width: 1.5;
    filter: drop-shadow(0 0 3px color-mix(in srgb, var(--accent-2) 60%, transparent));
  }
  .btn {
    fill: #1b1924;
    stroke: color-mix(in srgb, var(--accent-2) 70%, #000);
    stroke-width: 1.2;
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
    font-family: var(--font-body);
    font-weight: 700;
    font-size: 9px;
    fill: var(--text);
    text-anchor: middle;
    pointer-events: none;
  }
</style>
