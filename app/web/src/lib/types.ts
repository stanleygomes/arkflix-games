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
  manual?: string
  video?: string
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
  kidsOk?: boolean
  remote?: boolean
}

export interface ScreensaverSettings {
  enabled: boolean
  minutes: number
  mode: 'snake' | 'showcase'
  rest: number // minutos de protetor até o descanso (0 = nunca)
}


export interface MediaItem {
  src: string
  caption: string
  kind: 'image' | 'pdf' | 'video'
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
  videos: boolean
  videoSound: boolean
}

export interface EmulatorPaths {
  retroarch: string
  cores: string
  ppsspp: string
  pcsx2: string
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

export interface StorageInfo {
  disk: string
  total: number
  free: number
  games: number
  gameCount: number
  saves: number
  app: number
  cache: number
  byConsole: { system: string; name: string; bytes: number; count: number }[]
  saveDirs: { label: string; path: string; bytes: number }[]
  remote: number
  remoteCount: number
}

export interface RemoteInfo {
  connected: boolean
  base: string
  user: string
  owner: string
  mine: boolean
  games: number
  updated: number
}

export interface EmulatorInfo {
  id: string
  name: string
  installed: boolean
  path: string
  version: string
}

export interface CoreInfo {
  id: string
  systems: string[]
  installed: boolean
}

export interface EmulatorsInfo {
  platform: string
  canInstall: boolean
  reason?: string
  emulators: EmulatorInfo[]
  cores: CoreInfo[]
  coresDir: string
  busy: boolean
}

export interface BTDevice {
  address: string
  name: string
  paired: boolean
  connected: boolean
  trusted: boolean
  gamepad: boolean
  battery: number // -1 = não sabe
}

export interface BTInfo {
  supported: boolean
  powered: boolean
  scanning: boolean
  reason?: string
  devices: BTDevice[]
}

export interface CheevosInfo {
  connected: boolean
  user: string
  hardcore: boolean
  since?: number
}

export interface Cheevo {
  id: number
  title: string
  description: string
  points: number
  badgeUrl: string
  unlocked: boolean
  hardcore: boolean
}

export interface GameCheevos {
  connected: boolean
  supported: boolean
  reason?: string
  raGameId: number
  title: string
  total: number
  unlocked: number
  points: number
  pointsUnlocked: number
  list: Cheevo[] | null
}

export interface UpdateInfo {
  current: string
  latest: string
  available: boolean
  notes: string
  url: string
  asset: string
  size: number
  state: 'idle' | 'downloading' | 'ready' | 'restarting' | 'error'
  progress: number
  error?: string
  checked: number // última consulta (unix; 0 = nunca)
  restart: boolean // reinicia sozinho quando terminar de baixar
}

export interface GuideInfo {
  game: string // jogo aberto ('' = nenhum)
  commands: boolean // salvar/carregar/capturar pelo guia
  started: number // quando abriu (unix)
  recording: boolean
  note: string
}

export interface MediaFile {
  id: string
  kind: 'image' | 'video'
  at: number
  name: string
}

export interface TaskEvent {
  id: string
  label: string
  state: 'running' | 'done' | 'error' | 'cancelled'
  progress: number
  message?: string
}

export interface Settings {
  libraryDirs: LibraryDir[]
  importDir: string
  fullscreen: boolean
  esdeDir: string
  covers: Covers
  ui: { sort: Record<string, string> | null; filters: Record<string, string[]> | null }
  a11y: Accessibility
  screensaver: ScreensaverSettings
  saves: SaveDirs
  emulators: EmulatorPaths
  autoUpdate: boolean
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
  kidsOk?: boolean
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
  profile: ProfileState
  nativePads: boolean
  platform: string
}

export interface Profile {
  id: string
  name: string
  avatar: string
  color: string
  kids: boolean
  hasPIN: boolean
  main: boolean
}

export interface ProfileState {
  profiles: Profile[]
  current: string
  choose: boolean
  setup: boolean
  avatars: string[]
  colors: string[]
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
