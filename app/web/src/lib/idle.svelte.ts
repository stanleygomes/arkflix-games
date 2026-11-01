// Inatividade: quanto tempo sem tecla, controle ou mouse. O protetor de tela
// liga por aqui; qualquer entrada só acorda (não executa a ação).

export const idle = $state({ active: false })

let last = Date.now()
let pointer: { x: number; y: number } | null = null

export function idleFor(): number {
  return Date.now() - last
}

/** Registra atividade. Devolve true se o protetor estava ligado (e desliga):
 * quem chamou deve engolir a entrada. */
export function wake(): boolean {
  last = Date.now()
  if (!idle.active) return false
  idle.active = false
  return true
}

export function sleep() {
  idle.active = true
  pointer = null
}

let started = false
/** Mouse e toque também contam como atividade. */
export function watchPointer() {
  if (started) return
  started = true
  const swallow = (ev: Event) => {
    ev.preventDefault()
    ev.stopPropagation()
  }
  window.addEventListener(
    'pointerdown',
    (ev) => {
      if (!wake()) return
      swallow(ev)
      // o clique que vem depois do toque que acordou não vale
      const eat = (c: Event) => swallow(c)
      window.addEventListener('click', eat, { capture: true, once: true })
      setTimeout(() => window.removeEventListener('click', eat, { capture: true }), 600)
    },
    { capture: true },
  )
  window.addEventListener(
    'pointermove',
    (ev) => {
      if (!idle.active) {
        last = Date.now()
        return
      }
      // a tela muda por baixo do mouse e o navegador inventa um "move": só
      // acorda com movimento de verdade
      if (!pointer) pointer = { x: ev.clientX, y: ev.clientY }
      else if (Math.abs(ev.clientX - pointer.x) + Math.abs(ev.clientY - pointer.y) > 12) wake()
    },
    { capture: true, passive: true },
  )
  window.addEventListener('wheel', () => wake(), { capture: true, passive: true })
}
