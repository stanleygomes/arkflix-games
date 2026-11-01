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
