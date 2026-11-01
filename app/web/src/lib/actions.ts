// Ações de jogo usadas em várias telas.
import { api } from './api'
import { current } from './nav'
import { play } from './sound'
import { game, putGame, toast } from './state.svelte'

/** Favorita/desfavorita e avisa. */
export async function toggleFavorite(id: string | undefined) {
  const g = id ? game(id) : undefined
  if (!g) return
  try {
    const updated = await api.setFavorite(g.id, !g.favorite)
    putGame(updated)
    play('toggle')
    toast(updated.favorite ? `${updated.name} nos favoritos` : `${updated.name} saiu dos favoritos`, 'ok', 2500)
  } catch (err) {
    toast((err as Error).message, 'error')
  }
}

/** Jogo do card em foco, se houver. */
export function focusedGameId(): string | undefined {
  return current()?.dataset.game
}
