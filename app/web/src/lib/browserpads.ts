// Controles pela Gamepad API do WebView (Windows e macOS; no Linux quem lê é o
// Go, pelo kernel). Lê o estado a cada quadro e vira os mesmos eventos de
// botão que o Go manda: nomes posicionais (south, east, l1...) e direcional
// também pelo analógico esquerdo.
import type { Family, Pad, PadEvent } from './types'

// layout "standard" da Gamepad API, na ordem dos índices
const STANDARD = ['south', 'east', 'west', 'north', 'l1', 'r1', 'l2', 'r2', 'select', 'start', 'l3', 'r3', 'up', 'down', 'left', 'right', 'home']
const STICK = 0.5

export function familyOf(id: string): Family {
  if (/xbox|xinput|045e/i.test(id)) return 'xbox'
  if (/playstation|dualsense|dualshock|wireless controller.*054c|054c|sony/i.test(id)) return 'playstation'
  if (/8bitdo|2dc8/i.test(id)) return '8bitdo'
  if (/nintendo|057e|pro controller|joy-con/i.test(id)) return 'nintendo'
  return 'generic'
}

/** Nome legível: tira o "(STANDARD GAMEPAD Vendor: ...)" do id. */
export function padName(id: string): string {
  return id.replace(/\s*\(.*$/, '').trim() || 'Controle'
}

type Snapshot = Record<string, { pressed: boolean; source?: string }>

interface GamepadLike {
  index: number
  id: string
  connected: boolean
  buttons: readonly { pressed: boolean; value: number }[]
  axes: readonly number[]
}

export function snapshot(gp: GamepadLike): Snapshot {
  const out: Snapshot = {}
  STANDARD.forEach((name, i) => {
    const b = gp.buttons[i]
    if (b) out[name] = { pressed: b.pressed || b.value > STICK }
  })
  const [x = 0, y = 0] = gp.axes
  for (const [name, on] of [['left', x < -STICK], ['right', x > STICK], ['up', y < -STICK], ['down', y > STICK]] as const) {
    out[`stick:${name}`] = { pressed: on, source: 'stick' }
  }
  return out
}

/** Eventos entre dois estados (só o que mudou). */
export function changes(pad: Pad, before: Snapshot, after: Snapshot): PadEvent[] {
  const out: PadEvent[] = []
  for (const [key, now] of Object.entries(after)) {
    if ((before[key]?.pressed ?? false) === now.pressed) continue
    const button = key.startsWith('stick:') ? key.slice(6) : key
    out.push({ pad: pad.index, name: pad.name, family: pad.family, button, pressed: now.pressed, ...(now.source ? { source: now.source } : {}) })
  }
  return out
}

/** Começa a ler os controles; devolve a função de parar. */
export function startBrowserPads(
  onEvent: (e: PadEvent) => void,
  onPads: (pads: Pad[]) => void,
  read: () => (GamepadLike | null)[] = () => (navigator.getGamepads?.() ?? []) as (GamepadLike | null)[],
): () => void {
  const state = new Map<number, Snapshot>()
  let known = ''
  const tick = () => {
    const pads: Pad[] = []
    for (const gp of read()) {
      if (!gp?.connected) continue
      const pad: Pad = { index: gp.index, name: padName(gp.id), family: familyOf(gp.id) }
      pads.push(pad)
      const after = snapshot(gp)
      for (const e of changes(pad, state.get(gp.index) ?? {}, after)) onEvent(e)
      state.set(gp.index, after)
    }
    const key = pads.map((p) => `${p.index}:${p.name}`).join('|')
    if (key !== known) {
      known = key
      for (const i of [...state.keys()]) if (!pads.some((p) => p.index === i)) state.delete(i)
      onPads(pads)
    }
  }
  const iv = setInterval(tick, 16)
  return () => clearInterval(iv)
}
