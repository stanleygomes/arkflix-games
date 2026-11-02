// Estado global da interface (Svelte 5 runes).
import { play } from './sound'
import type { Boot, Family, Game, GameEvent, GameList, Mapping, Pad, ProfileState, Settings, System, RetroPadButton, UpdateInfo } from './types'

export type Screen =
  | { name: 'home' }
  | { name: 'game'; id: string }
  | { name: 'import' }
  | { name: 'search' }
  | { name: 'library'; source: string }
  | { name: 'edit'; id: string }
  | { name: 'stats' }
  | { name: 'settings'; tab?: SettingsTab }
  | { name: 'profiles' }
  | { name: 'profile'; id?: string; setup?: boolean; suggest?: string }
  | { name: 'welcome' }
  | { name: 'you' }
  | { name: 'activity' }

export type SettingsTab = 'library' | 'server' | 'storage' | 'emulators' | 'bluetooth' | 'updates' | 'cheevos' | 'controls' | 'a11y' | 'about'

export interface Toast {
  id: number
  text: string
  kind: 'info' | 'ok' | 'error'
}

/** Aviso guardado na central (os toasts somem; isto fica). */
export interface Notice {
  id: number
  text: string
  kind: Toast['kind']
  at: number
  read: boolean
  group: string // avisos do mesmo grupo seguidos viram um só ("3 avisos de download")
  count: number
}

export interface Confirm {
  text: string
  ok: string
  danger: boolean
  resolve: (v: boolean) => void
}

export interface PinAsk {
  title: string
  error: string
  submit: (pin: string) => Promise<string | null>
  resolve: (pin: string | null) => void
}

export interface Prompt {
  title: string
  value: string
  ok: string
  error: string
  max: number
  multiline: boolean
  secret: boolean
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
  notices: [] as Notice[],
  update: null as UpdateInfo | null,
  restarting: '', // versão pra qual o app está reiniciando
  guide: false,
  confirm: null as Confirm | null,
  prompt: null as Prompt | null,
  pin: null as PinAsk | null,
  power: false,
  // direção da última navegação (as telas animam de acordo)
  nav: { dir: 'fwd' as 'fwd' | 'back' | 'reset', seq: 0 },
  profile: null as ProfileState | null,
  platform: 'linux',
  lists: [] as GameList[],
  homeTab: 'home',
  selected: '',
  error: '',
})

export function screen(): Screen {
  return app.stack[app.stack.length - 1]
}

// telas que o perfil infantil não abre (a trava de verdade fica no Go)
const ADULT_ONLY = new Set<Screen['name']>(['settings', 'import', 'edit', 'profile'])

export function go(s: Screen) {
  if (isKids() && ADULT_ONLY.has(s.name)) {
    toast('Bloqueado no perfil infantil', 'info', 2500)
    return
  }
  app.nav = { dir: 'fwd', seq: app.nav.seq + 1 }
  app.stack.push(s)
}

/** Perfil em uso é infantil? */
export function isKids(): boolean {
  return !!currentProfile()?.kids
}

export function back(): boolean {
  if (app.stack.length > 1) {
    app.nav = { dir: 'back', seq: app.nav.seq + 1 }
    app.stack.pop()
    return true
  }
  return false
}

let nextToast = 1
export function toast(text: string, kind: Toast['kind'] = 'info', ms = 5000, group = '') {
  const id = nextToast++
  app.toasts.push({ id, text, kind })
  notice(id, text, kind, group)
  if (kind !== 'error') play('notify') // o erro já toca o som dele onde acontece
  setTimeout(() => {
    const i = app.toasts.findIndex((t) => t.id === id)
    if (i >= 0) app.toasts.splice(i, 1)
  }, ms)
}

// guarda na central; aviso do mesmo grupo logo em seguida (10 min) junta
function notice(id: number, text: string, kind: Toast['kind'], group: string) {
  const top = app.notices[0]
  if (group && top && top.group === group && !top.read && Date.now() - top.at < 600_000) {
    Object.assign(top, { text, kind: top.kind === 'error' ? 'error' : kind, at: Date.now(), count: top.count + 1 })
    return
  }
  app.notices.unshift({ id, text, kind, at: Date.now(), read: false, group, count: 1 })
  app.notices.splice(50)
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

/** Jogos que aparecem na biblioteca (sem os ocultos pelo perfil). */
export function shown(): Game[] {
  return app.games.filter((g) => !g.hidden)
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
export function prompt(
  title: string,
  value: string,
  ok: string,
  submit: (v: string) => Promise<string | null>,
  opts: { max?: number; multiline?: boolean; secret?: boolean } = {},
): Promise<string | null> {
  return new Promise((resolve) => {
    app.prompt = { title, value, ok, error: '', submit, resolve, max: opts.max ?? 60, multiline: !!opts.multiline, secret: !!opts.secret }
  })
}

/** Pede um PIN de 4 números. submit confere: devolve erro ou null. */
export function askPIN(title: string, submit: (pin: string) => Promise<string | null>): Promise<string | null> {
  return new Promise((resolve) => {
    app.pin = { title, error: '', submit, resolve }
  })
}

/** Põe o estado inicial (do app ou de um perfil) na interface. */
export function applyBoot(boot: Boot) {
  Object.assign(app, {
    version: boot.version,
    mode: boot.mode,
    systems: boot.systems,
    games: boot.games,
    settings: boot.settings,
    input: boot.input,
    retroPad: boot.retroPad,
    pads: boot.pads ?? [],
    lists: boot.lists ?? [],
    profile: boot.profile,
    platform: boot.platform || 'linux',
  })
}

/** Perfil em uso. */
export function currentProfile() {
  return app.profile?.profiles.find((p) => p.id === app.profile?.current)
}

/** Recomeça a navegação numa tela (troca de perfil). */
export function resetTo(s: Screen) {
  app.nav = { dir: 'reset', seq: app.nav.seq + 1 }
  app.stack = [s]
  app.homeTab = 'home'
  app.selected = ''
}

export function putList(l: GameList) {
  const i = app.lists.findIndex((x) => x.id === l.id)
  if (i >= 0) app.lists[i] = l
  else app.lists.push(l)
}
