// Entrada unificada: teclado e controle viram ações. A tela no topo da pilha
// de handlers decide primeiro; o padrão é navegar (setas), clicar (confirmar)
// e voltar. Na tela de controles, "captura" pega o próximo botão/tecla cru.
import { api, on } from './api'
import { wake } from './idle.svelte'
import { keyAction, padAction, type Action } from './keys'
import { current, move, type Dir } from './nav'
import { play, unlock } from './sound'
import { app, back, go, screen } from './state.svelte'
import type { PadEvent } from './types'

export type Source = 'pad' | 'key'
type Handler = (a: Action, src: Source) => boolean | void
const handlers: Handler[] = []

/** Registra o handler da tela; devolve a função pra remover. */
export function pushHandler(h: Handler): () => void {
  handlers.push(h)
  return () => {
    const i = handlers.lastIndexOf(h)
    if (i >= 0) handlers.splice(i, 1)
  }
}

export type Captured = { kind: 'pad'; button: string; family: PadEvent['family'] } | { kind: 'key'; code: string }
let capture: ((c: Captured) => void) | null = null

/** Próximo botão/tecla vai pra fn em vez de navegar (null cancela). */
export function setCapture(fn: ((c: Captured) => void) | null) {
  capture = fn
}

export function capturing(): boolean {
  return capture !== null
}

const listeners = new Set<(e: PadEvent) => void>()
/** Ouve os botões crus do controle (preview ao vivo). */
export function onPad(fn: (e: PadEvent) => void): () => void {
  listeners.add(fn)
  return () => listeners.delete(fn)
}

export function dispatch(action: Action, src: Source = 'key') {
  unlock()
  if (wake()) return
  if (app.playing?.state === 'running') return
  for (let i = handlers.length - 1; i >= 0; i--) {
    if (handlers[i](action, src) === true) return
  }
  switch (action) {
    case 'up':
    case 'down':
    case 'left':
    case 'right':
      if (move(action as Dir)) play('move')
      return
    case 'accept': {
      const el = current()
      if (el) {
        if (el.dataset.sfx !== 'none') play('accept')
        el.click()
      }
      return
    }
    case 'back':
      if (back()) play('back')
      return
    case 'search':
      if (screen().name !== 'search') {
        play('accept')
        go({ name: 'search' })
      }
      return
  }
}

const DIRS = new Set(['up', 'down', 'left', 'right'])
const repeat = new Map<string, number>()

function stopRepeat(key: string) {
  const t = repeat.get(key)
  if (t !== undefined) {
    clearTimeout(t)
    clearInterval(t)
    repeat.delete(key)
  }
}

function startRepeat(key: string, action: Action) {
  stopRepeat(key)
  const first = window.setTimeout(() => {
    const iv = window.setInterval(() => dispatch(action, 'pad'), 90)
    repeat.set(key, iv)
  }, 380)
  repeat.set(key, first)
}

function handlePad(e: PadEvent) {
  listeners.forEach((fn) => fn(e))
  if (e.pressed) app.family = e.family
  if (wake()) return // protetor de tela: o botão só acorda
  if (capture && e.pressed && !e.source) {
    const fn = capture
    capture = null
    fn({ kind: 'pad', button: e.button, family: e.family })
    return
  }
  if (capture) return
  const key = `${e.pad}:${e.button}:${e.source ?? ''}`
  if (!e.pressed) {
    stopRepeat(key)
    return
  }
  const action = padAction(e.button)
  if (!action) return
  dispatch(action, 'pad')
  if (DIRS.has(action)) startRepeat(key, action)
}

function handleKey(ev: KeyboardEvent) {
  app.family = 'keyboard'
  if (wake()) {
    ev.preventDefault()
    return
  }
  if (capture) {
    ev.preventDefault()
    const fn = capture
    capture = null
    fn({ kind: 'key', code: ev.code })
    return
  }
  if (ev.key === 'F11') {
    ev.preventDefault()
    api.toggleFullscreen().catch(() => {})
    return
  }
  const target = ev.target as HTMLElement | null
  const typing = !!target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)
  const action = keyAction(ev.key, typing)
  if (!action) return
  // Enter num campo envia o formulário normalmente
  if (typing && action === 'accept') return
  ev.preventDefault()
  dispatch(action, 'key')
}

let started = false
export function startInput() {
  if (started) return
  started = true
  window.addEventListener('keydown', handleKey)
  window.addEventListener('pointerdown', unlock, { once: true })
  on<PadEvent>('pad', handlePad)
}
