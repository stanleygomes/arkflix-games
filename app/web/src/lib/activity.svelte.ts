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

/** Ainda não terminou (em andamento, na fila, esperando ou pausado). */
export function active(t: TaskEvent): boolean {
  return t.state === 'running' || t.state === 'queued' || t.state === 'waiting' || t.state === 'paused'
}

/** O que dá pra cancelar: downloads (também na fila e pausados) e instalações. */
export function cancellable(t: TaskEvent): boolean {
  if (t.id.startsWith('download:')) return active(t)
  return t.state === 'running' && /^emulators:/.test(t.id + ':')
}

function upsert(ev: TaskEvent) {
  const i = activity.tasks.findIndex((t) => t.id === ev.id)
  if (i >= 0) activity.tasks.splice(i, 1)
  activity.tasks.unshift({ ...ev, at: Date.now() })
  activity.tasks.splice(30) // guarda as 30 últimas
}

let started = false
export function watchTasks() {
  if (started) return
  started = true
  // downloads pausados de antes de fechar o app (e o que já estiver na fila)
  api
    .downloads()
    .then((list) => {
      for (const ev of list.reverse()) if (!activity.tasks.some((t) => t.id === ev.id)) upsert(ev)
    })
    .catch(() => {})
  on<TaskEvent>('task', (ev) => {
    // progresso não reordena a lista (a fila fica parada na tela)
    const cur = activity.tasks.find((t) => t.id === ev.id)
    if (cur && cur.state === ev.state) Object.assign(cur, ev, { at: Date.now() })
    else upsert(ev)
    if (active(ev)) return
    const group = ev.id.split(':')[0]
    if (ev.state === 'done') toast(`${ev.label}: ${ev.message}`, 'ok', 7000, group)
    else if (ev.state === 'error') toast(`${ev.label} falhou: ${ev.message}`, 'error', 9000, group)
    else toast(`${ev.label}: cancelado`, 'info', 4000, group)
  })
}

async function attempt(fn: () => Promise<unknown>) {
  try {
    await fn()
  } catch (err) {
    toast((err as Error).message, 'error')
  }
}

export const cancelTask = (id: string) => attempt(() => api.cancelTask(id))

/** Pausar, continuar e "baixar agora" de um download (id da tarefa). */
export const pauseDownload = (id: string) => attempt(() => api.pauseDownload(id.replace(/^download:/, '')))
export const resumeDownload = (id: string) => attempt(() => api.resumeDownload(id.replace(/^download:/, '')))
export const downloadNow = (id: string) => attempt(() => api.downloadNow(id.replace(/^download:/, '')))

/** Some com as terminadas da lista. */
export function clearFinished() {
  activity.tasks = activity.tasks.filter(active)
}

/** Avisos ainda não vistos (bolinha no avatar e no guia). */
export function unread(): number {
  return app.notices.filter((n) => !n.read).length
}

export function markRead() {
  for (const n of app.notices) n.read = true
}
