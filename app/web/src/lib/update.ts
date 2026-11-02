// Atualização do app: estado global (aviso no topo, aba Atualizações) e o
// "Atualizar e reiniciar" num clique.
import { api, isDesktop, on } from './api'
import { play } from './sound'
import { app, toast } from './state.svelte'
import type { UpdateInfo } from './types'

/** "v1.9.3 → v1.10.0" (a versão de desenvolvimento vira só o número base). */
export function versionArrow(u: UpdateInfo): string {
  return `${shortVersion(u.current)} → ${u.latest}`
}

export function shortVersion(v: string): string {
  return /^v?\d+\.\d+\.\d+/.exec(v)?.[0] ?? v
}

let warned = ''
let started = false
export function watchUpdate() {
  if (started) return
  started = true
  api.updateStatus().then(set).catch(() => {})
  on<UpdateInfo>('update', set)
  on<string>('restart', (tag) => {
    app.restarting = tag || app.update?.latest || 'nova versão'
    // no navegador não tem app pra reabrir: recarrega a página
    if (!isDesktop()) setTimeout(() => location.reload(), 1500)
  })
}

function set(u: UpdateInfo) {
  app.update = u
  if (u.available && u.state !== 'restarting' && warned !== u.latest) {
    warned = u.latest
    toast(`Atualização disponível: ${versionArrow(u)}`, 'info', 8000, 'update')
  }
  if (u.state === 'error') app.restarting = '' // não instalou: o app segue aberto
}

/** Um clique: baixa se faltar, instala e reabre o app. */
export async function updateNow() {
  try {
    await api.updateNow()
    play('accept')
  } catch (err) {
    play('error')
    toast((err as Error).message, 'error')
  }
}
