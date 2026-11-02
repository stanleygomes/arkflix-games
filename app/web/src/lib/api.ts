// Transporte da API. No desktop, o Wails injeta window.go.core.Service (cada
// método do Go vira uma Promise) e window.runtime (eventos). No navegador, os
// mesmos métodos vão por POST /api/call/<Método> e os eventos por SSE.
// Trocar pra um servidor remoto no futuro = mudar só a URL base daqui.
import type {
  Boot,
  Game,
  GameList,
  BTInfo,
  CheevosInfo,
  EmulatorsInfo,
  GameCheevos,
  UpdateInfo,
  GuideInfo,
  ImportItem,
  ImportResult,
  InputResult,
  Listing,
  Mapping,
  MetaEdit,
  MetaInfo,
  MoveTarget,
  Pad,
  Profile,
  ProfileState,
  RemoteInfo,
  StorageInfo,
  ScriptsInfo,
  Settings,
  Stats,
} from './types'

type Service = Record<string, (...args: unknown[]) => Promise<unknown>>

interface WailsWindow {
  go?: { core?: { Service?: Service } }
  runtime?: {
    EventsOn: (name: string, cb: (data: unknown) => void) => (() => void) | void
    EventsOff: (name: string) => void
  }
}

const w = window as unknown as WailsWindow

export const isDesktop = (): boolean => typeof w.runtime?.EventsOn === 'function'

async function binding(): Promise<Service> {
  for (let i = 0; i < 100; i++) {
    const svc = w.go?.core?.Service
    if (svc) return svc
    await new Promise((r) => setTimeout(r, 50))
  }
  throw new Error('ligação com o Go não ficou pronta')
}

export async function call<T>(method: string, ...args: unknown[]): Promise<T> {
  if (isDesktop()) {
    const svc = await binding()
    const fn = svc[method]
    if (!fn) throw new Error(`método ${method} não existe`)
    try {
      return (await fn(...args)) as T
    } catch (err) {
      throw new Error(typeof err === 'string' ? err : String((err as Error)?.message ?? err))
    }
  }
  const res = await fetch(`/api/call/${method}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-Arkflix': '1' },
    body: JSON.stringify(args),
  })
  const body = (await res.json().catch(() => ({}))) as { result?: T; error?: string }
  if (!res.ok) throw new Error(body.error ?? `HTTP ${res.status}`)
  return body.result as T
}

type Listener = (data: unknown) => void
const listeners = new Map<string, Set<Listener>>()
let source: EventSource | null = null

function ensureSource() {
  if (source || isDesktop()) return
  source = new EventSource('/api/events')
  source.onmessage = (msg) => {
    const { name, data } = JSON.parse(msg.data) as { name: string; data: unknown }
    listeners.get(name)?.forEach((fn) => fn(data))
  }
}

/** Assina um evento do Go ("pad", "pads", "game", "library"). Devolve o cancelamento. */
export function on<T>(name: string, fn: (data: T) => void): () => void {
  const cb = fn as Listener
  if (isDesktop()) {
    const off = w.runtime!.EventsOn(name, cb)
    return typeof off === 'function' ? off : () => w.runtime!.EventsOff(name)
  }
  ensureSource()
  if (!listeners.has(name)) listeners.set(name, new Set())
  listeners.get(name)!.add(cb)
  return () => listeners.get(name)?.delete(cb)
}

export const api = {
  bootstrap: () => call<Boot>('Bootstrap'),
  profiles: () => call<ProfileState>('Profiles'),
  selectProfile: (id: string, pin: string) => call<Boot>('SelectProfile', id, pin),
  createProfile: (name: string, avatar: string, color: string) => call<Profile>('CreateProfile', name, avatar, color),
  updateProfile: (id: string, name: string, avatar: string, color: string) => call<Profile>('UpdateProfile', id, name, avatar, color),
  setProfilePIN: (id: string, current: string, next: string) => call<Profile>('SetProfilePIN', id, current, next),
  checkProfilePIN: (id: string, pin: string) => call<void>('CheckProfilePIN', id, pin),
  deleteProfile: (id: string, pin: string) => call<void>('DeleteProfile', id, pin),
  setProfileKids: (id: string, kids: boolean) => call<Profile>('SetProfileKids', id, kids),
  remoteStatus: () => call<RemoteInfo>('RemoteStatus'),
  storage: () => call<StorageInfo>('Storage'),
  movePlan: (system: string, from: string) => call<MoveTarget[]>('MovePlan', system, from),
  moveGames: (system: string, from: string, to: string) => call<void>('MoveGames', system, from, to),
  connectRemote: (host: string, user: string, pass: string) => call<RemoteInfo>('ConnectRemote', host, user, pass),
  disconnectRemote: () => call<void>('DisconnectRemote'),
  refreshRemote: () => call<void>('RefreshRemote'),
  downloadGame: (id: string) => call<void>('DownloadGame', id),
  ready: () => call<void>('Ready'),
  refresh: () => call<Game[]>('Refresh'),
  setFavorite: (id: string, fav: boolean) => call<Game>('SetFavorite', id, fav),
  play: (id: string, state = '') => call<void>('Play', id, state),
  saves: (id: string) => call<Listing>('Saves', id),
  deleteState: (id: string, state: string) => call<Listing>('DeleteState', id, state),
  openManual: (id: string) => call<void>('OpenManual', id),
  analyzeImport: (paths: string[]) => call<ImportItem[]>('AnalyzeImport', paths),
  importItems: (items: ImportItem[]) => call<ImportResult[]>('Import', items),
  pickFiles: () => call<string[] | null>('PickFiles'),
  getSettings: () => call<Settings>('GetSettings'),
  saveSettings: (s: Settings) => call<Settings>('SaveSettings', s),
  resetSettings: () => call<Settings>('ResetSettings'),
  getInput: () => call<Mapping>('GetInput'),
  saveInput: (m: Mapping) => call<InputResult>('SaveInput', m),
  resetInput: () => call<InputResult>('ResetInput'),
  pads: () => call<Pad[]>('Pads'),
  scripts: () => call<ScriptsInfo>('Scripts'),
  updateCovers: (all = false) => call<void>('UpdateCovers', all),
  lists: () => call<GameList[]>('Lists'),
  createList: (name: string) => call<GameList>('CreateList', name),
  renameList: (id: string, name: string) => call<GameList>('RenameList', id, name),
  deleteList: (id: string) => call<void>('DeleteList', id),
  setInList: (id: string, game: string, inList: boolean) => call<GameList>('SetInList', id, game, inList),
  metaDetails: (id: string) => call<MetaInfo>('MetaDetails', id),
  stats: (period: string) => call<Stats>('Stats', period),
  saveMeta: (id: string, edit: MetaEdit) => call<Game>('SaveMeta', id, edit),
  resetMeta: (id: string) => call<Game>('ResetMeta', id),
  toggleFullscreen: () => call<void>('ToggleFullscreen'),
  quit: () => call<void>('Quit'),
  minimize: () => call<void>('Minimize'),
  isFullscreen: () => call<boolean>('IsFullscreen'),
  cancelTask: (id: string) => call<void>('CancelTask', id),
  emulatorStatus: () => call<EmulatorsInfo>('EmulatorStatus'),
  installEmulators: (which: string[] = []) => call<void>('InstallEmulators', which),
  btStatus: () => call<BTInfo>('BluetoothStatus'),
  btScan: (seconds = 20) => call<void>('BluetoothScan', seconds),
  btPower: (on: boolean) => call<void>('BluetoothPower', on),
  btPair: (addr: string) => call<void>('BluetoothPair', addr),
  btForget: (addr: string) => call<void>('BluetoothForget', addr),
  btConnect: (addr: string) => call<void>('BluetoothConnect', addr),
  btDisconnect: (addr: string) => call<void>('BluetoothDisconnect', addr),
  btSettings: () => call<void>('OpenBluetoothSettings'),
  cheevosStatus: () => call<CheevosInfo>('CheevosStatus'),
  cheevosLogin: (user: string, pass: string) => call<CheevosInfo>('CheevosLogin', user, pass),
  cheevosLogout: () => call<void>('CheevosLogout'),
  gameCheevos: (id: string) => call<GameCheevos>('GameCheevos', id),
  updateStatus: () => call<UpdateInfo>('UpdateStatus'),
  checkUpdate: () => call<UpdateInfo>('CheckUpdate'),
  downloadUpdate: () => call<void>('DownloadUpdate'),
  updateNow: () => call<void>('UpdateNow'),
  guide: () => call<GuideInfo>('Guide'),
  showGuide: () => call<void>('ShowGuide'),
  resumeGame: () => call<void>('ResumeGame'),
  gameCommand: (cmd: 'save' | 'load' | 'quit') => call<void>('GameCommand', cmd),
}
