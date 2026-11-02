// Central de atividade: tarefas do Go (downloads, capas, saves, emuladores,
// atualização) e o histórico dos avisos. A barra flutuante, o guia e a tela
// "Central" leem daqui.
import { api, on } from './api'
import { app, toast } from './state.svelte'
import type { TaskEvent } from './types'

export interface Task extends TaskEvent {
  at: number // última mudança (ms)
}

export const activity = $state({ tasks: [] as Task[] })

/** Em andamento primeiro, depois as terminadas mais novas. */
export function running(): Task[] {
  return activity.tasks.filter((t) => t.state === 'running')
}

/** O que dá pra cancelar: downloads, instalações e a busca de informações. */
export function cancellable(t: TaskEvent): boolean {
  return t.state === 'running' && /^(download|emulators|meta):/.test(t.id + ':')
}

let started = false
export function watchTasks() {
  if (started) return
  started = true
  on<TaskEvent>('task', (ev) => {
    const i = activity.tasks.findIndex((t) => t.id === ev.id)
    const t = { ...ev, at: Date.now() }
    if (i >= 0) activity.tasks.splice(i, 1)
    activity.tasks.unshift(t)
    activity.tasks.splice(30) // guarda as 30 últimas
    if (ev.state === 'running') return
    const group = ev.id.split(':')[0]
    if (ev.state === 'done') toast(`${ev.label}: ${ev.message}`, 'ok', 7000, group)
    else if (ev.state === 'error') toast(`${ev.label} falhou: ${ev.message}`, 'error', 9000, group)
    else toast(`${ev.label}: cancelado`, 'info', 4000, group)
  })
}

export async function cancelTask(id: string) {
  try {
    await api.cancelTask(id)
  } catch (err) {
    toast((err as Error).message, 'error')
  }
}

/** Some com as terminadas da lista. */
export function clearFinished() {
  activity.tasks = activity.tasks.filter((t) => t.state === 'running')
}

/** Avisos ainda não vistos (bolinha no avatar e no guia). */
export function unread(): number {
  return app.notices.filter((n) => !n.read).length
}

export function markRead() {
  for (const n of app.notices) n.read = true
}
