import { describe, expect, it } from 'vitest'
import { ago, bytes, playTime } from '../../src/lib/format'
import { glyph, keyAction, keyLabel, padAction, retroArchKey } from '../../src/lib/keys'
import { filterGames, filterGroups, maxPlayers, normalize, pickRandom, search, sortGames } from '../../src/lib/library'
import type { Game, GameList, System } from '../../src/lib/types'
import { shortcuts } from '../../src/lib/shortcuts'
import { pickTarget, type Box } from '../../src/lib/nav'
import { contrast, highContrastPalette, homePalette, onColor } from '../../src/lib/palette'

const box = (left: number, top: number, w = 100, h = 50): Box => ({ left, top, right: left + w, bottom: top + h })

describe('contraste', () => {
  it('calcula a razão WCAG', () => {
    expect(contrast('#000000', '#ffffff')).toBeCloseTo(21, 1)
    expect(contrast('#777777', '#777777')).toBeCloseTo(1, 5)
  })
  it.each([
    ['inicial', homePalette],
    ['alto contraste', highContrastPalette],
  ])('paleta %s passa no AA', (_, p) => {
    expect(contrast(p.text, p.bg)).toBeGreaterThanOrEqual(4.5)
    expect(contrast(p.muted, p.bg)).toBeGreaterThanOrEqual(4.5)
    expect(contrast(p.text, p.surface)).toBeGreaterThanOrEqual(4.5)
    expect(contrast(p.accent, p.bg)).toBeGreaterThanOrEqual(3)
  })
  it('alto contraste chega no AAA', () => {
    expect(contrast(highContrastPalette.text, highContrastPalette.bg)).toBeGreaterThanOrEqual(7)
  })
  it('escolhe texto legível sobre o destaque', () => {
    expect(onColor('#ffd400')).toBe('#111111')
    expect(onColor('#1d1d6b')).toBe('#e8fbff')
    for (const bg of ['#ff9b54', '#a99bff', '#5ec4f2', '#ff5cc6']) expect(contrast(onColor(bg), bg)).toBeGreaterThanOrEqual(4.5)
  })
})

describe('navegação espacial', () => {
  const row = [box(0, 0), box(120, 0), box(240, 0), box(360, 0)]
  it('anda na fileira pro vizinho', () => {
    expect(pickTarget(row[1], [row[0], row[2], row[3]], 'right')).toBe(1)
    expect(pickTarget(row[1], [row[0], row[2], row[3]], 'left')).toBe(0)
  })
  it('não sai da fileira na horizontal se não tiver nada', () => {
    expect(pickTarget(row[3], row.slice(0, 3), 'right')).toBe(-1)
  })
  it('desce pro alinhado, não pro mais perto em diagonal', () => {
    const cur = box(240, 0)
    const below = [box(0, 100), box(230, 140), box(480, 100)]
    expect(pickTarget(cur, below, 'down')).toBe(1)
  })
  it('sobe pro elemento acima mesmo desalinhado se for o único', () => {
    expect(pickTarget(box(500, 300), [box(0, 0, 80, 40)], 'up')).toBe(0)
  })
  it('na grade escolhe a coluna certa', () => {
    const grid = [box(0, 0), box(120, 0), box(0, 70), box(120, 70)]
    expect(pickTarget(grid[1], [grid[0], grid[2], grid[3]], 'down')).toBe(2)
  })
})

describe('entrada', () => {
  it('controle: confirma embaixo, volta à direita', () => {
    expect(padAction('south')).toBe('accept')
    expect(padAction('east')).toBe('back')
    expect(padAction('north')).toBe('fav')
    expect(padAction('l1')).toBe('prevTab')
    expect(padAction('select')).toBe('search')
    expect(padAction('home')).toBe('guide')
    expect(padAction('l3')).toBeNull()
  })
  it('teclado: setas e Enter; num campo de texto só Esc e cima/baixo', () => {
    expect(keyAction('ArrowLeft', false)).toBe('left')
    expect(keyAction('Enter', false)).toBe('accept')
    expect(keyAction('f', false)).toBe('fav')
    expect(keyAction('ArrowLeft', true)).toBeNull()
    expect(keyAction('f', true)).toBeNull()
    expect(keyAction(' ', true)).toBeNull()
    expect(keyAction('Escape', true)).toBe('back')
    expect(keyAction('ArrowDown', true)).toBe('down')
  })
  it('traduz tecla pro nome do RetroArch', () => {
    expect(retroArchKey('KeyZ')).toBe('z')
    expect(retroArchKey('Digit1')).toBe('num1')
    expect(retroArchKey('ShiftRight')).toBe('rshift')
    expect(retroArchKey('F2')).toBe('f2')
    expect(retroArchKey('F13')).toBeNull()
    expect(retroArchKey('MetaLeft')).toBeNull()
    for (const code of ['KeyA', 'Digit9', 'ArrowUp', 'Enter', 'Space', 'Numpad5', 'BracketLeft']) {
      expect(retroArchKey(code)).toMatch(/^[a-z0-9_]{1,16}$/)
    }
  })
  it('mostra o nome da tecla', () => {
    expect(keyLabel('rshift')).toBe('Shift dir.')
    expect(keyLabel('num1')).toBe('1')
    expect(keyLabel('z')).toBe('Z')
  })
  it('símbolos por controle', () => {
    expect(glyph('south', 'xbox')).toBe('A')
    expect(glyph('south', 'playstation')).toBe('✕')
    expect(glyph('south', '8bitdo')).toBe('B')
    expect(glyph('east', 'nintendo')).toBe('A')
    expect(glyph('up', 'playstation')).toBe('↑')
    expect(glyph('south', 'keyboard')).toBe('Enter')
  })
})

describe('formatação', () => {
  it('tempo de jogo', () => {
    expect(playTime(0)).toBe('nunca jogado')
    expect(playTime(30)).toBe('menos de 1 min')
    expect(playTime(125)).toBe('2 min')
    expect(playTime(3600 * 2 + 60 * 5)).toBe('2 h 5 min')
  })
  it('tempo relativo', () => {
    const now = 1_000_000
    expect(ago(0, now)).toBe('nunca')
    expect(ago(now - 10, now)).toBe('agora')
    expect(ago(now - 600, now)).toBe('há 10 min')
    expect(ago(now - 86400 - 10, now)).toBe('ontem')
  })
  it('tamanho', () => {
    expect(bytes(512)).toBe('512 B')
    expect(bytes(1536)).toBe('1,5 KB')
    expect(bytes(4.7 * 1024 ** 3)).toBe('4,7 GB')
  })
})

const fakeGame = (name: string, extra: Partial<Game> = {}): Game => ({
  id: name, system: 'snes', title: name, fileName: `${name}.zip`, folder: 'snes', root: '/', local: true, size: 1, added: 0, hasSaves: false,
  name, summary: '', year: 0, genres: [], players: '', cover: '', snap: '', titleShot: '', favorite: false,
  lastPlayed: 0, playTime: 0, plays: 0, ...extra,
})

describe('busca', () => {
  const games = ['Super Mario World', 'Super Mario 64', 'Pokémon Gold', 'Mega Man X', 'Mario Kart 64'].map((n) => fakeGame(n))
  it('normaliza acento, maiúscula e pontuação', () => {
    expect(normalize('Pokémon: Gold!')).toBe('pokemon gold')
    expect(normalize('Tom & Jerry')).toBe('tom and jerry')
  })
  it('acha por parte do nome, sem acento', () => {
    expect(search(games, 'mario').map((g) => g.name)).toEqual(['Mario Kart 64', 'Super Mario 64', 'Super Mario World'])
    expect(search(games, 'POKEMON').map((g) => g.name)).toEqual(['Pokémon Gold'])
    expect(search(games, 'mario 64').map((g) => g.name)).toEqual(['Mario Kart 64', 'Super Mario 64'])
  })
  it('acha pelo nome do arquivo e ignora busca vazia', () => {
    const g = fakeGame('Sonic 2', { fileName: 'Sonic The Hedgehog 2 (World).zip' })
    expect(search([g], 'hedgehog')).toHaveLength(1)
    expect(search(games, '   ')).toEqual([])
  })
})

describe('ordenação', () => {
  const games = [
    fakeGame('Beta', { year: 1995, plays: 2, lastPlayed: 100, playTime: 50, added: 3 }),
    fakeGame('alfa', { year: 1990, plays: 5, lastPlayed: 300, playTime: 10, added: 1 }),
    fakeGame('Gama', { year: 0, plays: 2, lastPlayed: 0, playTime: 900, added: 2 }),
  ]
  const order = (key: string) => sortGames(games, key).map((g) => g.name)
  it.each([
    ['name', ['alfa', 'Beta', 'Gama']],
    ['yearAsc', ['alfa', 'Beta', 'Gama']],
    ['yearDesc', ['Beta', 'alfa', 'Gama']],
    ['plays', ['alfa', 'Beta', 'Gama']],
    ['recent', ['alfa', 'Beta', 'Gama']],
    ['playTime', ['Gama', 'Beta', 'alfa']],
    ['added', ['Beta', 'Gama', 'alfa']],
    ['desconhecida', ['alfa', 'Beta', 'Gama']],
  ])('%s', (key, want) => {
    expect(order(key)).toEqual(want)
  })
  it('não altera a lista original', () => {
    sortGames(games, 'playTime')
    expect(games[0].name).toBe('Beta')
  })
})

describe('filtros', () => {
  const games = [
    fakeGame('A', { genres: ['Plataforma'], players: '1-2', favorite: true, year: 1990, hasSaves: true, lastPlayed: 5 }),
    fakeGame('B', { genres: ['Esporte'], players: '1', year: 1998 }),
    fakeGame('C', { genres: ['Plataforma', 'Ação'], players: '1-4', year: 1993, completed: true, lastPlayed: 9 }),
  ]
  const names = (f: string[]) => filterGames(games, f).map((g) => g.name)
  it('jogadores', () => {
    expect(maxPlayers('1-4')).toBe(4)
    expect(maxPlayers('')).toBe(0)
    expect(names(['players:2'])).toEqual(['A', 'C'])
    expect(names(['players:1'])).toEqual(['B'])
  })
  it('mesmo grupo soma, grupos diferentes combinam', () => {
    expect(names(['genre:Esporte', 'genre:Ação'])).toEqual(['B', 'C'])
    expect(names(['genre:Plataforma', 'decade:1990'])).toEqual(['A', 'C'])
    expect(names(['genre:Plataforma', 'players:2', 'fav'])).toEqual(['A'])
    expect(names(['never'])).toEqual(['B'])
    expect(names(['saves'])).toEqual(['A'])
    expect(names(['completed'])).toEqual(['C'])
    expect(names([])).toEqual(['A', 'B', 'C'])
  })
  it('grupos só com o que existe', () => {
    const groups = filterGroups(games)
    expect(groups.map((g) => g.id)).toEqual(['quick', 'players', 'genre', 'decade'])
    expect(groups[2].options.map((o) => o.label)).toEqual(['Ação', 'Esporte', 'Plataforma'])
    expect(groups[3].options.map((o) => o.id)).toEqual(['decade:1990'])
    expect(filterGroups([fakeGame('X', { lastPlayed: 1 })])).toEqual([])
  })
})

describe('sorteio', () => {
  const games = ['A', 'B', 'C'].map((n) => fakeGame(n))
  it('nunca repete o último quando há outra opção', () => {
    for (let i = 0; i < 50; i++) expect(pickRandom(games, 'B')?.id).not.toBe('B')
    expect(pickRandom([fakeGame('Só')], 'Só')?.id).toBe('Só')
  })
  it('respeita a lista dada e cobre todos', () => {
    const seen = new Set<string>()
    for (let i = 0; i < 200; i++) seen.add(pickRandom(games)!.id)
    expect([...seen].sort()).toEqual(['A', 'B', 'C'])
    expect(pickRandom([], 'x')).toBeUndefined()
    expect(pickRandom(games, '', () => 0.999)?.id).toBe('C')
  })
})

describe('busca global', () => {
  const systems = [
    { slug: 'snes', name: 'Super Nintendo', short: 'SNES' },
    { slug: 'nes', name: 'Nintendo Entertainment System', short: 'NES' },
  ] as unknown as System[]
  const ctx = { systems, withGames: new Set(['snes']), lists: [{ id: 'a', name: 'Plataforma raiz', games: ['x'] }] as GameList[], kids: false, desktop: false }
  it('acha configuração por palavra parecida e sem acento', () => {
    expect(shortcuts('espaço', ctx).map((s) => s.label)).toEqual(['Armazenamento'])
    expect(shortcuts('tela cheia', ctx)).toEqual([]) // energia só no desktop
    expect(shortcuts('tela cheia', { ...ctx, desktop: true }).map((s) => s.label)).toEqual(['Energia'])
  })
  it('console só com jogo; lista pelo nome; infantil não vê configuração', () => {
    expect(shortcuts('nintendo', ctx).map((s) => s.id)).toEqual(['system:snes'])
    expect(shortcuts('plataf', ctx).map((s) => s.kind)).toEqual(['list'])
    expect(shortcuts('bluetooth', { ...ctx, kids: true })).toEqual([])
  })
})
