// Navegação espacial (como TV/console): a direção escolhe o elemento
// focável mais próximo naquele sentido. Focáveis têm [data-nav]; um
// [data-nav-scope] (diálogo) prende o foco dentro dele. [data-nav-group]
// lembra o último item focado do grupo (descer pra fileira volta no jogo
// selecionado, como no PS5).

export type Dir = 'up' | 'down' | 'left' | 'right'

export interface Box {
  left: number
  top: number
  right: number
  bottom: number
}

const TOL = 4

/** Escolhe o índice do melhor candidato (ou -1). Função pura: testável. */
export function pickTarget(cur: Box, candidates: Box[], dir: Dir): number {
  const cx = (cur.left + cur.right) / 2
  const cy = (cur.top + cur.bottom) / 2
  let best = -1
  let bestScore = Infinity
  candidates.forEach((b, i) => {
    const bx = (b.left + b.right) / 2
    const by = (b.top + b.bottom) / 2
    let primary: number
    let secondary: number
    switch (dir) {
      case 'right':
        if (b.left < cur.right - TOL && bx <= cx + TOL) return
        primary = Math.max(0, b.left - cur.right)
        secondary = Math.abs(by - cy)
        break
      case 'left':
        if (b.right > cur.left + TOL && bx >= cx - TOL) return
        primary = Math.max(0, cur.left - b.right)
        secondary = Math.abs(by - cy)
        break
      case 'down':
        if (b.top < cur.bottom - TOL && by <= cy + TOL) return
        primary = Math.max(0, b.top - cur.bottom)
        secondary = Math.abs(bx - cx)
        break
      case 'up':
        if (b.bottom > cur.top + TOL && by >= cy - TOL) return
        primary = Math.max(0, cur.top - b.bottom)
        secondary = Math.abs(bx - cx)
        break
    }
    // fora da faixa (sem sobreposição no outro eixo) pesa mais: prefere alinhado
    const horizontal = dir === 'left' || dir === 'right'
    const overlap = horizontal
      ? Math.min(cur.bottom, b.bottom) - Math.max(cur.top, b.top)
      : Math.min(cur.right, b.right) - Math.max(cur.left, b.left)
    const score = primary + secondary * (overlap > 0 ? 1 : 3)
    if (score < bestScore) {
      bestScore = score
      best = i
    }
  })
  return best
}

const SELECTOR = '[data-nav]:not([disabled]):not([aria-hidden="true"])'
const memory = new Map<string, HTMLElement>()

function visible(el: HTMLElement): boolean {
  if (el.closest('[inert]')) return false // tela ou diálogo saindo
  const r = el.getBoundingClientRect()
  return r.width > 0 && r.height > 0 && getComputedStyle(el).visibility !== 'hidden'
}

function scope(): ParentNode {
  const dialogs = [...document.querySelectorAll<HTMLElement>('[data-nav-scope]')].filter((d) => !d.closest('[inert]'))
  return dialogs.length ? dialogs[dialogs.length - 1] : document
}

export function focusables(): HTMLElement[] {
  return [...scope().querySelectorAll<HTMLElement>(SELECTOR)].filter(visible)
}

let reduced = false
export function setReducedMotion(v: boolean) {
  reduced = v
}

export function focusEl(el: HTMLElement | null | undefined): boolean {
  if (!el) return false
  el.focus({ preventScroll: true })
  el.scrollIntoView({
    block: 'nearest',
    inline: el.dataset.navInline === 'center' ? 'center' : 'nearest',
    behavior: reduced ? 'auto' : 'smooth',
  })
  const group = el.closest<HTMLElement>('[data-nav-group]')?.dataset.navGroup
  if (group) memory.set(group, el)
  return true
}

/** Foca o padrão de dentro de el (painel embutido na tela Você). */
export function focusWithin(el: HTMLElement | null | undefined): boolean {
  if (!el) return false
  return focusEl(el.querySelector<HTMLElement>('[data-nav-default]') ?? el.querySelector<HTMLElement>('[data-nav]'))
}

/** Foca o padrão da tela ([data-nav-default]) ou o primeiro focável. */
export function focusDefault(): boolean {
  const s = scope()
  const def = [...s.querySelectorAll<HTMLElement>('[data-nav-default]')].find(visible)
  if (def) return focusEl(def)
  return focusEl(focusables()[0])
}

export function current(): HTMLElement | null {
  const el = document.activeElement as HTMLElement | null
  if (!el || el === document.body) return null
  const nav = el.closest<HTMLElement>('[data-nav]')
  return nav && scope().contains(nav) ? nav : null
}

/** Move o foco na direção. Devolve false se não tinha pra onde ir. */
export function move(dir: Dir): boolean {
  const cur = current()
  if (!cur) return focusDefault()
  const all = focusables().filter((el) => el !== cur)
  const idx = pickTarget(cur.getBoundingClientRect(), all.map((el) => el.getBoundingClientRect()), dir)
  if (idx < 0) return false
  let target = all[idx]
  const group = target.closest<HTMLElement>('[data-nav-group]')?.dataset.navGroup
  const curGroup = cur.closest<HTMLElement>('[data-nav-group]')?.dataset.navGroup
  if (group && group !== curGroup) {
    const remembered = memory.get(group)
    if (remembered && remembered.isConnected && visible(remembered)) target = remembered
  }
  return focusEl(target)
}

export function forgetGroup(group: string) {
  memory.delete(group)
}
