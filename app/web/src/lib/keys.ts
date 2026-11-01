// Traduções de entrada: tecla/botão -> ação da interface, tecla -> nome do
// RetroArch, botão físico -> símbolo do controle. Tudo puro (testável).
import type { Family } from './types'

export type Action =
  | 'up'
  | 'down'
  | 'left'
  | 'right'
  | 'accept'
  | 'back'
  | 'fav'
  | 'details'
  | 'menu'
  | 'prevTab'
  | 'nextTab'

/** Botão físico (nomes do Go) -> ação. Confirma com o botão de baixo. */
export function padAction(button: string): Action | null {
  const map: Record<string, Action> = {
    up: 'up',
    down: 'down',
    left: 'left',
    right: 'right',
    south: 'accept',
    east: 'back',
    north: 'fav',
    west: 'details',
    start: 'menu',
    l1: 'prevTab',
    r1: 'nextTab',
  }
  return map[button] ?? null
}

/** Tecla -> ação. typing = foco num campo de texto (setas e espaço são do campo). */
export function keyAction(key: string, typing: boolean): Action | null {
  if (typing) {
    if (key === 'Escape') return 'back'
    if (key === 'ArrowUp') return 'up'
    if (key === 'ArrowDown') return 'down'
    return null
  }
  const map: Record<string, Action> = {
    ArrowUp: 'up',
    ArrowDown: 'down',
    ArrowLeft: 'left',
    ArrowRight: 'right',
    Enter: 'accept',
    ' ': 'accept',
    Escape: 'back',
    Backspace: 'back',
    f: 'fav',
    F: 'fav',
    i: 'details',
    I: 'details',
    m: 'menu',
    M: 'menu',
    PageUp: 'prevTab',
    PageDown: 'nextTab',
    q: 'prevTab',
    e: 'nextTab',
  }
  return map[key] ?? null
}

/** KeyboardEvent.code -> nome de tecla do RetroArch (ou null se não suportada). */
export function retroArchKey(code: string): string | null {
  const letter = /^Key([A-Z])$/.exec(code)
  if (letter) return letter[1].toLowerCase()
  const digit = /^Digit(\d)$/.exec(code)
  if (digit) return 'num' + digit[1]
  const fkey = /^F(\d{1,2})$/.exec(code)
  if (fkey && Number(fkey[1]) <= 12) return 'f' + fkey[1]
  const pad = /^Numpad(\d)$/.exec(code)
  if (pad) return 'keypad' + pad[1]
  const named: Record<string, string> = {
    ArrowUp: 'up',
    ArrowDown: 'down',
    ArrowLeft: 'left',
    ArrowRight: 'right',
    Enter: 'enter',
    NumpadEnter: 'kp_enter',
    Space: 'space',
    Tab: 'tab',
    Backspace: 'backspace',
    ShiftLeft: 'shift',
    ShiftRight: 'rshift',
    ControlLeft: 'ctrl',
    ControlRight: 'rctrl',
    AltLeft: 'alt',
    AltRight: 'ralt',
    Insert: 'insert',
    Delete: 'del',
    Home: 'home',
    End: 'end',
    PageUp: 'pageup',
    PageDown: 'pagedown',
    Comma: 'comma',
    Period: 'period',
    Slash: 'slash',
    Semicolon: 'semicolon',
    Minus: 'minus',
    Equal: 'equals',
    BracketLeft: 'leftbracket',
    BracketRight: 'rightbracket',
    Backslash: 'backslash',
    Quote: 'quote',
    Backquote: 'tilde',
  }
  return named[code] ?? null
}

/** Nome legível de uma tecla do RetroArch. */
export function keyLabel(name: string): string {
  const labels: Record<string, string> = {
    up: '↑',
    down: '↓',
    left: '←',
    right: '→',
    enter: 'Enter',
    kp_enter: 'Enter (num)',
    space: 'Espaço',
    rshift: 'Shift dir.',
    shift: 'Shift',
    ctrl: 'Ctrl',
    rctrl: 'Ctrl dir.',
    alt: 'Alt',
    ralt: 'AltGr',
    backspace: 'Backspace',
    tab: 'Tab',
    escape: 'Esc',
  }
  if (labels[name]) return labels[name]
  if (/^num\d$/.test(name)) return name.slice(3)
  if (/^keypad\d$/.test(name)) return 'Num ' + name.slice(6)
  return name.toUpperCase()
}

/** Símbolo do botão físico em cada família de controle. */
export function glyph(button: string, family: Family): string {
  const face: Record<string, Record<string, string>> = {
    xbox: { south: 'A', east: 'B', west: 'X', north: 'Y', l1: 'LB', r1: 'RB', l2: 'LT', r2: 'RT', select: 'View', start: 'Menu', home: 'Xbox' },
    playstation: { south: '✕', east: '○', west: '□', north: '△', l1: 'L1', r1: 'R1', l2: 'L2', r2: 'R2', select: 'Create', start: 'Options', home: 'PS' },
    '8bitdo': { south: 'B', east: 'A', west: 'Y', north: 'X', l1: 'L', r1: 'R', l2: 'L2', r2: 'R2', select: 'Select', start: 'Start', home: 'Home' },
    nintendo: { south: 'B', east: 'A', west: 'Y', north: 'X', l1: 'L', r1: 'R', l2: 'ZL', r2: 'ZR', select: '−', start: '+', home: 'Home' },
    keyboard: { south: 'Enter', east: 'Esc', west: 'I', north: 'F', l1: 'Q', r1: 'E', start: 'M' },
  }
  const common: Record<string, string> = { up: '↑', down: '↓', left: '←', right: '→', l3: 'L3', r3: 'R3' }
  const table = face[family] ?? face.xbox
  return table[button] ?? common[button] ?? button
}
