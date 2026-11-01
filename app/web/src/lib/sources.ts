// Fontes de lista: console, coleção automática ou lista pessoal. A chave
// ("system:snes", "collection:all") guarda a ordem e os filtros de cada uma.
import { app, system } from './state.svelte'
import type { Game } from './types'

export interface Collection {
  id: string
  label: string
  short: string
  pick: (g: Game) => boolean
  sort: string
}

export const COLLECTIONS: Collection[] = [
  { id: 'all', label: 'Todos os jogos', short: 'Todos', pick: () => true, sort: 'name' },
  { id: 'favorites', label: 'Favoritos', short: 'Favoritos', pick: (g) => g.favorite, sort: 'name' },
  { id: 'recent', label: 'Continuar jogando', short: 'Recentes', pick: (g) => g.lastPlayed > 0, sort: 'recent' },
  { id: 'most', label: 'Mais jogados', short: 'Mais jogados', pick: (g) => g.plays > 0, sort: 'plays' },
  { id: 'never', label: 'Nunca jogados', short: 'Nunca jogados', pick: (g) => !g.lastPlayed, sort: 'name' },
  { id: 'completed', label: 'Completos', short: 'Completos', pick: (g) => !!g.completed, sort: 'name' },
  { id: 'server', label: 'No servidor', short: 'Servidor', pick: (g) => !!g.remote, sort: 'name' },
]

export function collection(id: string): Collection | undefined {
  return COLLECTIONS.find((c) => c.id === id)
}

/** Coleções com pelo menos um jogo (vazia não aparece). */
export function activeCollections(games: Game[]): Collection[] {
  return COLLECTIONS.filter((c) => games.some(c.pick))
}

export function sourceTitle(source: string): string {
  const [kind, id] = source.split(':')
  if (kind === 'system') return system(id)?.name ?? id
  if (kind === 'collection') return collection(id)?.label ?? id
  if (kind === 'list') return app.lists.find((l) => l.id === id)?.name ?? 'Lista'
  return id
}

export function sourceGames(source: string): Game[] {
  const [kind, id] = source.split(':')
  if (kind === 'system') return app.games.filter((g) => g.system === id)
  if (kind === 'collection') {
    const c = collection(id)
    return c ? app.games.filter(c.pick) : []
  }
  if (kind === 'list') {
    // jogo apagado do disco some da lista sem erro
    const ids = new Set(app.lists.find((l) => l.id === id)?.games ?? [])
    return app.games.filter((g) => ids.has(g.id))
  }
  return []
}

export function filtersOf(source: string): string[] {
  return app.settings?.ui?.filters?.[source] ?? []
}

export function sortOf(source: string): string {
  const saved = app.settings?.ui?.sort?.[source]
  if (saved) return saved
  const [kind, id] = source.split(':')
  return kind === 'collection' ? (collection(id)?.sort ?? 'name') : 'name'
}
