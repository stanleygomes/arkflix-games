<script module lang="ts">
  // Consoles em pixel art (desenho próprio, não os logotipos oficiais, que
  // são marcas registradas). Cada letra do desenho é uma cor da paleta.
  type Art = { colors: Record<string, string>; rows: string[] }

  const ART: Record<string, Art> = {
    home: {
      colors: { a: '#ff3cac', b: '#050008', c: '#ffd23f' },
      rows: [
        '................', '.......aa.......', '......aaaa......', '.....aaaaaa.....', '....aaaaaaaa....', '...aaaaaaaaaa...',
        '..aaaaaaaaaaaa..', '...aaaaaaaaaa...', '...aabbaaccaa...', '...aabbaaccaa...', '...aabbaaaaaa...', '...aabbaaaaaa...',
        '...aaaaaaaaaa...', '................', '................', '................',
      ],
    },
    nes: {
      colors: { a: '#b8b8bc', b: '#2a2a2e', c: '#e02828', d: '#e8e8ec' },
      rows: [
        '................', '................', '................', '................', 'dddddddddddddddd', 'aaaaaaaaaaaaaaaa',
        'abbbbbbbbbbbbbba', 'aaaaaaaaaaaaaaaa', 'bbbbbbbbbbbbbbbb', 'bddbddbbbbbbbbbb', 'bbbbbbbbbbbbbcbb', 'bbbbbbbbbbbbbbbb',
        'bbbbbbbbbbbbbbbb', '................', '................', '................',
      ],
    },
    snes: {
      colors: { a: '#c8c8d2', b: '#5a5a70', c: '#8a5ad8', d: '#ececf4' },
      rows: [
        '................', '................', '................', '.....bbbbbb.....', '....abbbbbba....', '.aaaaaaaaaaaaaa.',
        'adddddddddddddda', 'aaaaaaaaaaaaaaaa', 'accaaaaaaaaaacca', 'accaaaaaaaaaacca', 'aaaaaaaaaaaaaaaa', 'abbbbbbbbbbbbbba',
        '.aaaaaaaaaaaaaa.', '................', '................', '................',
      ],
    },
    genesis: {
      colors: { a: '#1e1e22', b: '#4a4a52', c: '#e02828', d: '#8a8a94' },
      rows: [
        '................', '................', '................', '..aaaaaaaaaaaa..', '.aaaaabbbbaaaaa.', 'aaaabddddddbaaaa',
        'aaabdccccccdbaaa', 'aaabdcbbbbcdbaaa', 'aaabdccccccdbaaa', 'aaaabddddddbaaaa', '.aaaaabbbbaaaaa.', '..aaaaaaaaaaaa..',
        '................', '................', '................', '................',
      ],
    },
    n64: {
      colors: { a: '#3a3a44', b: '#18181c', c: '#28b048', d: '#6a6a78' },
      rows: [
        '................', '.....dddddd.....', '.....dbbbbd.....', '.....dbbbbd.....', '..aaaaaaaaaaaa..', '.aaaaaaaaaaaaaa.',
        'aaaaaaaaaaaaaaaa', 'aaaaaaaaaaaaaaaa', 'acaaaaaaaaaaaaaa', 'aaaaaaaaaaaaaaaa', '.aaaaaaaaaaaaaa.', '..aa........aa..',
        '................', '................', '................', '................',
      ],
    },
    psx: {
      colors: { a: '#c4c4cc', b: '#8a8a96', c: '#3a5ad0', d: '#e4e4ec' },
      rows: [
        '................', '................', 'aaaaaaaaaaaaaaaa', 'aaaaabbbbbbaaaaa', 'aaaabddddddbaaaa', 'aaabddddddddbaaa',
        'aaabddddddddbaaa', 'aaaabddddddbaaaa', 'aaaaabbbbbbaaaaa', 'aaaaaaaaaaaaaaaa', 'acaaaaaaaaaabbba', 'aaaaaaaaaaaaaaaa',
        '................', '................', '................', '................',
      ],
    },
    ps2: {
      colors: { a: '#1c1c26', b: '#3a3a4c', c: '#3a6aff', d: '#5a5a72' },
      rows: [
        '.....aaaaaa.....', '.....abbbba.....', '.....aaaaaa.....', '.....abbbba.....', '.....aaaaaa.....', '.....aaaaaa.....',
        '.....adddda.....', '.....aaaaaa.....', '.....aaaaaa.....', '.....aaaaaa.....', '.....aaaaaa.....', '.....aaaaca.....',
        '.....aaaaaa.....', '....bbbbbbbb....', '................', '................',
      ],
    },
    psp: {
      colors: { a: '#202028', b: '#050508', c: '#6a8aff', d: '#6a6a7a' },
      rows: [
        '................', '................', '................', '................', '..aaaaaaaaaaaa..', '.aaabbbbbbbbaaa.',
        'adaabccccccbaada', 'addabccccccbadda', 'adaabccccccbaada', '.aaabbbbbbbbaaa.', '..aaaaaaaaaaaa..', '................',
        '................', '................', '................', '................',
      ],
    },
    gbc: {
      colors: { a: '#8a4ad8', b: '#1c1c22', c: '#9ac85a', d: '#e04080' },
      rows: [
        '....aaaaaaaa....', '....abbbbbba....', '....abccccba....', '....abccccba....', '....abccccba....', '....abbbbbba....',
        '....aaaaaaaa....', '....abaaaaaa....', '....bbbaadaa....', '....abaadaaa....', '....aaaaaaaa....', '....aaaaaaaa....',
        '.....aaaaaa.....', '................', '................', '................',
      ],
    },
    nds: {
      colors: { a: '#c8c8d0', b: '#202024', c: '#7aa8ff', d: '#8a8a96' },
      rows: [
        '...aaaaaaaaaa...', '...abbbbbbbba...', '...abccccccba...', '...abccccccba...', '...abbbbbbbba...', '...aaaaaaaaaa...',
        '...dddddddddd...', '...aaaaaaaaaa...', '...abbbbbbbba...', '...abccccccba...', '...abbbbbbbba...', '...aaaaaaaaaa...',
        '................', '................', '................', '................',
      ],
    },
    arcade: {
      colors: { a: '#2a3ab0', b: '#050508', c: '#ff3cac', d: '#ffd23f' },
      rows: [
        '...aaaaaaaaaa...', '...adddddddda...', '...aaaaaaaaaa...', '...abbbbbbbba...', '...abccccccba...', '...abccccccba...',
        '...abbbbbbbba...', '..aaaaaaaaaaaa..', '..aadaacaacaaa..', '...aaaaaaaaaa...', '...aaaaaaaaaa...', '...aaaaaaaaaa...',
        '...aaaaaaaaaa...', '...aaaaaaaaaa...', '................', '................',
      ],
    },
  }
  ART.gb = ART.gbc
  ART.gba = ART.gbc
</script>

<script lang="ts">
  let { slug, size = '3rem' }: { slug: string; size?: string } = $props()
  const art = $derived(ART[slug] ?? ART.arcade)
  // linhas desenhadas do topo; centraliza na vertical
  const used = $derived(art.rows.map((r, y) => (r.replace(/\./g, '') ? y : -1)).filter((y) => y >= 0))
  const offset = $derived(used.length ? Math.floor((16 - (used[used.length - 1] - used[0] + 1)) / 2) - used[0] : 0)
  const pixels = $derived(
    art.rows.flatMap((row, y) => [...row].map((ch, x) => ({ x, y: y + offset, fill: art.colors[ch] })).filter((p) => p.fill)),
  )
</script>

<svg viewBox="0 0 16 16" width={size} height={size} shape-rendering="crispEdges" aria-hidden="true">
  {#each pixels as p, i (i)}<rect x={p.x} y={p.y} width="1" height="1" fill={p.fill} />{/each}
</svg>
