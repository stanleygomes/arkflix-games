// Espelho das structs do Go (internal/core, settings, saves...). Mudou lá, muda aqui.

export interface Palette {
  bg: string
  surface: string
  accent: string
  accent2: string
  text: string
  muted: string
}

export interface System {
  slug: string
  name: string
  short: string
  maker: string
  year: number
  media: 'cartridge' | 'disc' | 'umd' | 'card' | 'arcade'
  exts: string[]
  emulator: string
  core?: string
  palette: Palette
}

export interface Game {
  id: string
  system: string
  title: string
  fileName: string
  folder: string
  root: string
  local: boolean
  size: number
  added: number
  name: string
  summary: string
  year: number
  genres: string[]
  players: string
  cover: string
  snap: string
  titleShot: string
  favorite: boolean
  lastPlayed: number
  playTime: number
  plays: number
  hasSaves: boolean
  completed?: boolean
  rating?: number
  broken?: boolean
  developer?: string
  edited?: boolean
}

export interface LibraryDir {
  path: string
  label: string
  writable: boolean
}

export interface Accessibility {
  fontScale: number
  highContrast: boolean
  reduceMotion: boolean
  scanlines: boolean
  pixelTitles: boolean
  sounds: boolean
  soundVolume: number
  music: boolean
  musicVolume: number
}

export interface SaveDirs {
  retroarchSaves: string
  retroarchStates: string
  ppssppSaveData: string
  ppssppStates: string
  pcsx2Memcards: string
  pcsx2States: string
}

export interface Covers {
  enabled: boolean
  images: string
  db: string
}

export interface TaskEvent {
  id: string
  label: string
  state: 'running' | 'done' | 'error'
  progress: number
  message?: string
}

export interface Settings {
  libraryDirs: LibraryDir[]
  importDir: string
  emulatorBin: string
  coresDir: string
  fullscreen: boolean
  esdeDir: string
  covers: Covers
  ui: { sort: Record<string, string> | null; filters: Record<string, string[]> | null }
  a11y: Accessibility
  saves: SaveDirs
}

export interface Mapping {
  pad: Record<string, string>
  keys: Record<string, string>
}

export interface RetroPadButton {
  id: string
  label: string
}

export type Family = 'xbox' | 'playstation' | '8bitdo' | 'nintendo' | 'generic' | 'keyboard'

export interface Pad {
  index: number
  name: string
  family: Family
}

export interface PadEvent {
  pad: number
  name: string
  family: Family
  button: string
  pressed: boolean
  source?: string
}

export interface MetaFields {
  name: string
  summary: string
  year: number
  genres: string[] | null
  players: string
  developer: string
}

export interface MetaEdit {
  name?: string
  summary?: string
  developer?: string
  players?: string
  year?: number
  genres?: string[]
  cover?: string
  rating: number
  completed: boolean
  broken: boolean
}

export interface MetaInfo {
  auto: MetaFields
  autoCover: boolean
  edit: MetaEdit
}

export interface Stats {
  period: string
  totalSeconds: number
  plays: number
  sessions: number
  gamesPlayed: number
  libraryGames: number
  byGame: { id: string; name: string; system: string; seconds: number; plays: number; last: number }[]
  bySystem: { system: string; seconds: number; plays: number; games: number; played: number }[]
  byWeekday: number[]
  byDay: { date: string; seconds: number }[]
  bestDay: { date: string; seconds: number }
}

export interface GameList {
  id: string
  name: string
  games: string[]
  created: number
}

export interface Boot {
  version: string
  mode: 'desktop' | 'web'
  systems: System[]
  games: Game[]
  settings: Settings
  input: Mapping
  retroPad: RetroPadButton[]
  pads: Pad[]
  running: string
  lists: GameList[]
}

export interface GameEvent {
  state: 'running' | 'exited'
  id: string
  name: string
  seconds?: number
  code?: number
  error?: string
}

export interface SaveState {
  id: string
  slot: number
  label: string
  modified: number
  thumb: boolean
}

export interface SaveFile {
  path: string
  name: string
  size: number
  modified: number
  shared: boolean
}

export interface Listing {
  states: SaveState[]
  files: SaveFile[]
  note?: string
}

export interface ImportItem {
  source: string
  name: string
  title: string
  system: string
  candidates: string[] | null
  extract: boolean
  size: number
  problem?: string
}

export interface ImportResult extends ImportItem {
  dest?: string
  ok: boolean
}

export interface ScriptsInfo {
  dir: string
  log: string
  events: Record<string, string[]>
}

export interface InputResult {
  mapping: Mapping
  conflicts: string[]
}
