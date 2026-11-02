// Busca global além dos jogos: consoles, listas e as telas/configurações
// ("tela cheia", "bluetooth", "espaço"...). Puro (testável).
import { normalize } from './library'
import type { Screen } from './state.svelte'
import type { GameList, System } from './types'

export interface Shortcut {
  id: string
  label: string
  hint: string
  kind: 'console' | 'list' | 'setting'
  go?: Screen
  power?: boolean // abre o menu de energia
}

type Entry = { label: string; hint: string; words: string; go?: Screen; power?: boolean; adult?: boolean; desktop?: boolean }

const SETTINGS: Entry[] = [
  { label: 'Controles', hint: 'mapear botões', words: 'controle controles botoes mapear joystick gamepad teclado', go: { name: 'settings', tab: 'controls' }, adult: true },
  { label: 'Acessibilidade e som', hint: 'texto, contraste, animações, música', words: 'acessibilidade texto fonte letra contraste animacao animacoes som sons musica volume video', go: { name: 'settings', tab: 'a11y' }, adult: true },
  { label: 'Protetor de tela', hint: 'cobrinha, vitrine e descanso', words: 'protetor tela cobrinha vitrine descanso', go: { name: 'settings', tab: 'a11y' }, adult: true },
  { label: 'Conquistas', hint: 'RetroAchievements', words: 'conquistas conquista trofeus trofeu retroachievements achievements', go: { name: 'settings', tab: 'cheevos' }, adult: true },
  { label: 'Biblioteca', hint: 'pastas de jogos e capas', words: 'biblioteca pasta pastas roms capas', go: { name: 'settings', tab: 'library' }, adult: true },
  { label: 'Servidor RomM', hint: 'jogos e saves na nuvem', words: 'servidor romm nuvem sincronizar sincronia saves', go: { name: 'settings', tab: 'server' }, adult: true },
  { label: 'Armazenamento', hint: 'espaço em disco', words: 'armazenamento espaco disco livre tamanho', go: { name: 'settings', tab: 'storage' }, adult: true },
  { label: 'Emuladores', hint: 'instalar e atualizar', words: 'emuladores emulador retroarch ppsspp pcsx2 cores core instalar', go: { name: 'settings', tab: 'emulators' }, adult: true },
  { label: 'Bluetooth', hint: 'parear controle sem fio', words: 'bluetooth parear pareamento sem fio wireless', go: { name: 'settings', tab: 'bluetooth' }, adult: true },
  { label: 'Sobre e atualização', hint: 'versão do app', words: 'sobre versao atualizar atualizacao update atalhos scripts', go: { name: 'settings', tab: 'about' }, adult: true },
  { label: 'Importar jogos', hint: 'de pasta ou pendrive', words: 'importar pendrive copiar adicionar jogo', go: { name: 'import' }, adult: true },
  { label: 'Estatísticas', hint: 'horas jogadas', words: 'estatisticas horas tempo jogado historico', go: { name: 'stats' } },
  { label: 'Trocar perfil', hint: 'perfis da casa', words: 'perfil perfis trocar usuario infantil pin', go: { name: 'profiles' } },
  { label: 'Central', hint: 'downloads e avisos', words: 'central downloads download avisos notificacoes notificacao', go: { name: 'activity' } },
  { label: 'Energia', hint: 'tela cheia, minimizar, fechar', words: 'energia tela cheia janela fullscreen minimizar fechar sair desligar', power: true, desktop: true },
]

export function shortcuts(
  query: string,
  ctx: { systems: System[]; withGames: Set<string>; lists: GameList[]; kids: boolean; desktop: boolean },
): Shortcut[] {
  const words = normalize(query).split(' ').filter(Boolean)
  if (!words.length) return []
  const hit = (hay: string) => words.every((w) => normalize(hay).split(' ').some((h) => h.startsWith(w)))
  const out: Shortcut[] = []
  for (const s of ctx.systems) {
    if (ctx.withGames.has(s.slug) && hit(`${s.name} ${s.short} ${s.slug}`)) {
      out.push({ id: `system:${s.slug}`, label: s.name, hint: 'console', kind: 'console', go: { name: 'library', source: `system:${s.slug}` } })
    }
  }
  for (const l of ctx.lists) {
    if (hit(l.name)) out.push({ id: `list:${l.id}`, label: l.name, hint: `lista · ${l.games.length} jogos`, kind: 'list', go: { name: 'library', source: `list:${l.id}` } })
  }
  for (const e of SETTINGS) {
    if ((e.adult && ctx.kids) || (e.desktop && !ctx.desktop)) continue
    if (hit(`${e.label} ${e.words}`)) out.push({ id: `setting:${e.label}`, label: e.label, hint: e.hint, kind: 'setting', go: e.go, power: e.power })
  }
  return out.slice(0, 8)
}
