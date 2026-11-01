<script lang="ts">
  // Avatar sobre a cor do perfil: personagem (imagem em /avatars) ou desenho
  // em pixel art 10x10 ("#" = claro, "o" = escuro).
  let { avatar, color, size = '6rem' }: { avatar: string; color: string; size?: string } = $props()

  const SPRITES: Record<string, string[]> = {
    knight: ['..######..', '.########.', '.#oooooo#.', '.#o#oo#o#.', '.#oooooo#.', '.########.', '...####...', '.########.', '##.####.##', '.#.#..#.#.'],
    mage: ['....##....', '...####...', '..##o###..', '.########.', '##########', '..#o##o#..', '..######..', '...#oo#...', '..######..', '.##.##.##.'],
    ghost: ['...####...', '.########.', '.########.', '##o##o####', '##o##o####', '##########', '##########', '##########', '##.##.##.#', '#...#...#.'],
    cat: ['#........#', '##......##', '##########', '#o##o#####', '##########', '###oo#####', '.########.', '..######..', '..#.##.#..', '..#.##.#..'],
    robot: ['....#.....', '....#.....', '.########.', '.#o####o#.', '.########.', '.#oooooo#.', '.########.', '##.####.##', '...####...', '...#..#...'],
    mushroom: ['...####...', '.##oo####.', '###oo##o##', '#######oo#', '#o########', '##########', '..#o##o#..', '..######..', '..######..', '...####...'],
    alien: ['..#....#..', '...#..#...', '..######..', '.##o##o##.', '##########', '#.######.#', '#.#....#.#', '...##.##..', '..........', '..........'],
    dragon: ['.......##.', '......####', '......#o##', '.....#####', '.##.####..', '########..', '.#######..', '..######..', '..#..#....', '.##.##....'],
  }

  const cells = $derived.by(() => {
    const out: { x: number; y: number; dark: boolean }[] = []
    ;(SPRITES[avatar] ?? []).forEach((row, y) =>
      [...row].forEach((c, x) => {
        if (c !== '.') out.push({ x, y, dark: c === 'o' })
      }),
    )
    return out
  })
</script>

<span class="avatar" style:background={color} style:width={size} style:height={size} aria-hidden="true">
  {#if SPRITES[avatar]}
    <svg viewBox="-1 -1 12 12" shape-rendering="crispEdges">
      {#each cells as c (c.y * 10 + c.x)}
        <rect x={c.x} y={c.y} width="1" height="1" fill={c.dark ? '#1a1423' : '#fff8e8'} />
      {/each}
    </svg>
  {:else}
    <img src={`/avatars/${avatar}.png`} alt="" draggable="false" />
  {/if}
</span>

<style>
  .avatar {
    display: inline-grid;
    place-items: center;
    flex: none;
    border: 3px solid rgba(0, 0, 0, 0.35);
    box-shadow: 4px 4px 0 var(--shadow);
  }
  svg {
    width: 78%;
    height: 78%;
  }
  img {
    width: 88%;
    height: 88%;
    object-fit: contain;
    image-rendering: pixelated; /* 64 px esticado: blocos, no clima 16-bit */
    filter: drop-shadow(2px 2px 0 rgba(0, 0, 0, 0.45));
  }
</style>
