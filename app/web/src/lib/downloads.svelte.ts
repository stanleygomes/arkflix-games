// Downloads de jogos do servidor em andamento (id do jogo -> 0..1), pelos
// eventos "task" do Go.
import { api, on } from './api'
import { play } from './sound'
import { toast } from './state.svelte'
import type { Game, TaskEvent } from './types'

export const downloads = $state<Record<string, number>>({})
/** Estado de cada download da fila (running, queued, waiting, paused). */
export const downloadState = $state<Record<string, TaskEvent['state']>>({})

/** Texto do botão/selo do jogo enquanto está na fila. */
export function downloadText(id: string): string {
  const pct = `${Math.round((downloads[id] ?? 0) * 100)}%`
  switch (downloadState[id]) {
    case 'queued':
      return 'Na fila'
    case 'waiting':
      return 'Esperando a madrugada'
    case 'paused':
      return `Pausado ${pct}`
  }
  return `Baixando ${pct}`
}

let started = false
export function watchDownloads() {
  if (started) return
  started = true
  on<TaskEvent>('task', (ev) => {
    if (!ev.id.startsWith('download:')) return
    const id = ev.id.slice('download:'.length)
    if (ev.state === 'running' || ev.state === 'queued' || ev.state === 'waiting' || ev.state === 'paused') {
      downloads[id] = ev.progress
      downloadState[id] = ev.state
    } else {
      delete downloads[id]
      delete downloadState[id]
    }
  })
  api
    .downloads()
    .then((list) => {
      for (const ev of list) {
        const id = ev.id.slice('download:'.length)
        downloads[id] = ev.progress
        downloadState[id] = ev.state
      }
    })
    .catch(() => {})
}

/** Começa a baixar o jogo do servidor (o fim chega pela biblioteca nova). */
export async function downloadGame(g: Game | undefined) {
  if (!g) return
  try {
    const busy = Object.keys(downloadState).length > 0
    await api.downloadGame(g.id)
    downloads[g.id] ??= 0
    play('accept')
    toast(busy ? `${g.name} entrou na fila (veja na Central)` : `Baixando ${g.name}…`, 'info', 2500, 'download')
  } catch (err) {
    play('error')
    toast((err as Error).message, 'error')
  }
}
