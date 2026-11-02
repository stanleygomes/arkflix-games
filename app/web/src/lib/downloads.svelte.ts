// Downloads de jogos do servidor em andamento (id do jogo -> 0..1), pelos
// eventos "task" do Go.
import { api, on } from './api'
import { play } from './sound'
import { toast } from './state.svelte'
import type { Game, TaskEvent } from './types'

export const downloads = $state<Record<string, number>>({})

let started = false
export function watchDownloads() {
  if (started) return
  started = true
  on<TaskEvent>('task', (ev) => {
    if (!ev.id.startsWith('download:')) return
    const id = ev.id.slice('download:'.length)
    if (ev.state === 'running') downloads[id] = ev.progress
    else delete downloads[id]
  })
}

/** Começa a baixar o jogo do servidor (o fim chega pela biblioteca nova). */
export async function downloadGame(g: Game | undefined) {
  if (!g) return
  try {
    await api.downloadGame(g.id)
    downloads[g.id] = 0
    play('accept')
    toast(`Baixando ${g.name}…`, 'info', 2500, 'download')
  } catch (err) {
    play('error')
    toast((err as Error).message, 'error')
  }
}
