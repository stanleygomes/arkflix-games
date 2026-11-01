// Fontes de lista: console, coleção ou lista pessoal. A chave ("system:snes")
// guarda a ordem e os filtros escolhidos pra cada uma.
import { app, system } from './state.svelte'
import type { Game } from './types'

export function sourceTitle(source: string): string {
  const [kind, id] = source.split(':')
  if (kind === 'system') return system(id)?.name ?? id
  return id
}

export function sourceGames(source: string): Game[] {
  const [kind, id] = source.split(':')
  if (kind === 'system') return app.games.filter((g) => g.system === id)
  return []
}

export function filtersOf(source: string): string[] {
  return app.settings?.ui?.filters?.[source] ?? []
}

export function sortOf(source: string): string {
  return app.settings?.ui?.sort?.[source] ?? 'name'
}
