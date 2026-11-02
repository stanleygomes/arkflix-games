// Busca, ordenação, filtros e sorteio da biblioteca. Funções puras (testáveis).
import type { Game } from './types'

/** Minúsculas, sem acento, só letras/números e espaço. */
export function normalize(s: string): string {
  return s
    .normalize('NFD')
    .replace(/\p{Mn}/gu, '')
    .toLowerCase()
    .replace(/&/g, ' and ')
    .replace(/[^a-z0-9]+/g, ' ')
    .trim()
}

/** Todas as palavras da busca precisam aparecer no nome, título ou arquivo. */
export function search(games: Game[], query: string): Game[] {
  const words = normalize(query).split(' ').filter(Boolean)
  if (!words.length) return []
  return games
    .map((g) => {
      const name = normalize(g.name)
      const hay = `${name} ${normalize(g.title)} ${normalize(g.fileName)}`
      if (!words.every((w) => hay.includes(w))) return null
      // começa com a busca vem antes; depois, palavra inteira; depois, o resto
      const score = name.startsWith(words.join(' ')) ? 0 : words.every((w) => name.split(' ').includes(w)) ? 1 : 2
      return { g, score }
    })
    .filter((x): x is { g: Game; score: number } => x !== null)
    .sort((a, b) => a.score - b.score || a.g.name.localeCompare(b.g.name, 'pt-BR'))
    .map((x) => x.g)
}

export type SortKey = 'name' | 'yearAsc' | 'yearDesc' | 'plays' | 'recent' | 'playTime' | 'rating' | 'added'

export const SORTS: { key: SortKey; label: string }[] = [
  { key: 'name', label: 'Nome (A–Z)' },
  { key: 'yearAsc', label: 'Ano (mais antigos)' },
  { key: 'yearDesc', label: 'Ano (mais novos)' },
  { key: 'plays', label: 'Mais jogados' },
  { key: 'recent', label: 'Jogados recentemente' },
  { key: 'playTime', label: 'Mais tempo de jogo' },
  { key: 'rating', label: 'Nota' },
  { key: 'added', label: 'Adicionados recentemente' },
]

const byName = (a: Game, b: Game) => a.name.localeCompare(b.name, 'pt-BR', { sensitivity: 'base' })

/** Ordena sem mexer na lista original; empate desempata pelo nome. Sem ano vai pro fim. */
export function sortGames(games: Game[], key: SortKey | string | undefined): Game[] {
  const rating = (g: Game) => g.rating ?? 0
  const cmp: Record<SortKey, (a: Game, b: Game) => number> = {
    name: () => 0,
    yearAsc: (a, b) => (a.year || 9999) - (b.year || 9999),
    yearDesc: (a, b) => (b.year || 0) - (a.year || 0),
    plays: (a, b) => b.plays - a.plays,
    recent: (a, b) => b.lastPlayed - a.lastPlayed,
    playTime: (a, b) => b.playTime - a.playTime,
    rating: (a, b) => rating(b) - rating(a),
    added: (a, b) => b.added - a.added,
  }
  const fn = cmp[(key as SortKey) in cmp ? (key as SortKey) : 'name']
  return [...games].sort((a, b) => fn(a, b) || byName(a, b))
}

/** Maior número de jogadores ("1-2" -> 2, "1" -> 1, "" -> 0). */
export function maxPlayers(players: string): number {
  const nums = (players.match(/\d+/g) ?? []).map(Number)
  return nums.length ? Math.max(...nums) : 0
}

export interface FilterGroup {
  id: string
  label: string
  options: { id: string; label: string }[]
}

/** Grupos de filtro possíveis pra esta lista (só o que tem jogo). */
export function filterGroups(games: Game[]): FilterGroup[] {
  const quick = [
    { id: 'fav', label: 'Favoritos', ok: games.some((g) => g.favorite) },
    { id: 'never', label: 'Nunca jogados', ok: games.some((g) => !g.lastPlayed) },
    { id: 'saves', label: 'Com save', ok: games.some((g) => g.hasSaves) },
    { id: 'completed', label: 'Completos', ok: games.some((g) => g.completed) },
  ].filter((q) => q.ok)
  const players = [
    { id: 'players:1', label: '1 jogador', ok: games.some((g) => maxPlayers(g.players) === 1) },
    { id: 'players:2', label: '2 ou mais', ok: games.some((g) => maxPlayers(g.players) >= 2) },
  ].filter((q) => q.ok)
  const genres = [...new Set(games.flatMap((g) => g.genres))].sort((a, b) => a.localeCompare(b, 'pt-BR'))
  const decades = [...new Set(games.filter((g) => g.year).map((g) => Math.floor(g.year / 10) * 10))].sort()
  return [
    { id: 'quick', label: 'Mostrar só', options: quick },
    { id: 'players', label: 'Jogadores', options: players },
    { id: 'genre', label: 'Gênero', options: genres.map((g) => ({ id: `genre:${g}`, label: g })) },
    { id: 'decade', label: 'Década', options: decades.map((d) => ({ id: `decade:${d}`, label: `Anos ${String(d).slice(2)} (${d})` })) },
  ].filter((g) => g.options.length)
}

function matches(g: Game, f: string): boolean {
  const [kind, value] = f.split(/:(.*)/s)
  switch (kind) {
    case 'fav':
      return g.favorite
    case 'never':
      return !g.lastPlayed
    case 'saves':
      return g.hasSaves
    case 'completed':
      return !!g.completed
    case 'players':
      return value === '1' ? maxPlayers(g.players) === 1 : maxPlayers(g.players) >= 2
    case 'genre':
      return g.genres.includes(value)
    case 'decade':
      return !!g.year && Math.floor(g.year / 10) * 10 === Number(value)
  }
  return true
}

/** Filtros do mesmo grupo somam (OU); grupos diferentes se combinam (E). */
export function filterGames(games: Game[], active: string[]): Game[] {
  if (!active.length) return games
  const groups = new Map<string, string[]>()
  for (const f of active) {
    const kind = f.includes(':') ? f.split(':')[0] : 'quick-' + f
    groups.set(kind, [...(groups.get(kind) ?? []), f])
  }
  return games.filter((g) => [...groups.values()].every((fs) => fs.some((f) => matches(g, f))))
}

/** Lista automática: os jogos da fonte, filtrados e ordenados pela regra. */
export function ruleGames(source: Game[], rule: { filters: string[]; sort: string }): Game[] {
  return sortGames(filterGames(source, rule.filters), rule.sort)
}

/** Sorteia um jogo sem repetir o último (se houver outra opção). */
export function pickRandom(games: Game[], lastId = '', rnd: () => number = Math.random): Game | undefined {
  const pool = games.length > 1 ? games.filter((g) => g.id !== lastId) : games
  if (!pool.length) return undefined
  return pool[Math.min(pool.length - 1, Math.floor(rnd() * pool.length))]
}
