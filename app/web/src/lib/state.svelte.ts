// Estado global da interface (Svelte 5 runes).
import type { Family, Game, GameEvent, GameList, Mapping, Pad, Settings, System, RetroPadButton } from './types'

export type Screen =
  | { name: 'home' }
  | { name: 'game'; id: string }
  | { name: 'import' }
  | { name: 'search' }
  | { name: 'library'; source: string }
  | { name: 'settings'; tab?: SettingsTab }

export type SettingsTab = 'library' | 'controls' | 'a11y' | 'about'

export interface Toast {
  id: number
  text: string
  kind: 'info' | 'ok' | 'error'
}

export interface Confirm {
  text: string
  ok: string
  danger: boolean
  resolve: (v: boolean) => void
}

export interface Prompt {
  title: string
  value: string
  ok: string
  error: string
  submit: (v: string) => Promise<string | null>
  resolve: (v: string | null) => void
}

export const app = $state({
  ready: false,
  version: '',
  mode: 'web' as 'web' | 'desktop',
  systems: [] as System[],
  games: [] as Game[],
  settings: null as Settings | null,
  input: null as Mapping | null,
  retroPad: [] as RetroPadButton[],
  pads: [] as Pad[],
  family: 'keyboard' as Family,
  stack: [{ name: 'home' }] as Screen[],
  playing: null as GameEvent | null,
  toasts: [] as Toast[],
  confirm: null as Confirm | null,
  prompt: null as Prompt | null,
  lists: [] as GameList[],
  homeTab: 'home',
  selected: '',
  error: '',
})

export function screen(): Screen {
  return app.stack[app.stack.length - 1]
}

export function go(s: Screen) {
  app.stack.push(s)
}

export function back(): boolean {
  if (app.stack.length > 1) {
    app.stack.pop()
    return true
  }
  return false
}

let nextToast = 1
export function toast(text: string, kind: Toast['kind'] = 'info', ms = 5000) {
  const id = nextToast++
  app.toasts.push({ id, text, kind })
  setTimeout(() => {
    const i = app.toasts.findIndex((t) => t.id === id)
    if (i >= 0) app.toasts.splice(i, 1)
  }, ms)
}

export function confirm(text: string, ok = 'Confirmar', danger = false): Promise<boolean> {
  return new Promise((resolve) => {
    app.confirm = { text, ok, danger, resolve }
  })
}

export function system(slug: string): System | undefined {
  return app.systems.find((s) => s.slug === slug)
}

export function game(id: string): Game | undefined {
  return app.games.find((g) => g.id === id)
}

export function putGame(g: Game) {
  const i = app.games.findIndex((x) => x.id === g.id)
  if (i >= 0) app.games[i] = g
}

/** Família de controle pros símbolos de botão (genérico usa os do Xbox). */
export function glyphFamily(): Family {
  return app.family === 'generic' ? 'xbox' : app.family
}

/** Pede um texto (com teclado na tela). submit valida/salva: devolve erro ou null. */
export function prompt(title: string, value: string, ok: string, submit: (v: string) => Promise<string | null>): Promise<string | null> {
  return new Promise((resolve) => {
    app.prompt = { title, value, ok, error: '', submit, resolve }
  })
}

export function putList(l: GameList) {
  const i = app.lists.findIndex((x) => x.id === l.id)
  if (i >= 0) app.lists[i] = l
  else app.lists.push(l)
}
