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
  const rating = (g: Game) => (g as Game & { rating?: number }).rating ?? 0
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
