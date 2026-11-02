// Animações de tela, diálogo e efeitos. Todas em degraus (steps), no clima
// 16-bit, curtas e desligadas com "reduzir animações" (do app ou do sistema).
// O que sai fica inerte: o foco do controle nunca cai numa tela de saída.

export function reduced(): boolean {
  return (
    document.documentElement.classList.contains('reduce-motion') ||
    (typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches)
  )
}

/** Arredonda o progresso em n degraus (cara de animação de 16 bits). */
export const stepped = (t: number, n: number) => Math.min(1, Math.floor(t * n) / n)

type Css = { duration: number; delay?: number; css?: (t: number, u: number) => string }

export type ScreenMotion = { kind: 'slide' | 'crt' | 'none'; dir: 1 | -1 }

const SCREEN_MS = 240

/** Entrada de tela: desliza na direção da navegação, ou liga como uma TV. */
export function screenIn(_node: Element, p: ScreenMotion): Css {
  if (reduced() || p.kind === 'none') return { duration: 0 }
  if (p.kind === 'crt') {
    return {
      duration: 320,
      css: (t) => {
        const s = stepped(t, 8)
        // linha fina que abre na vertical e brilha antes de assentar
        const y = s < 0.4 ? 0.01 : (s - 0.4) / 0.6
        return `transform: scaleX(${0.6 + 0.4 * Math.min(1, s * 2)}) scaleY(${Math.max(0.01, y)}); filter: brightness(${1 + (1 - s) * 2.5});`
      },
    }
  }
  return {
    duration: SCREEN_MS,
    css: (t) => `transform: translateX(${(1 - stepped(t, 6)) * 22 * p.dir}%); opacity: ${stepped(t, 6)};`,
  }
}

/** Saída de tela: some pro lado oposto (ou apaga), e fica inerte. */
export function screenOut(node: Element, p: ScreenMotion): Css {
  ;(node as HTMLElement).inert = true
  if (reduced() || p.kind === 'none') return { duration: 0 }
  return {
    duration: SCREEN_MS,
    css: (t) => `transform: translateX(${(1 - stepped(t, 6)) * -16 * p.dir}%); opacity: ${stepped(t, 6)};`,
  }
}

/** Diálogo: nasce de uma linha neon que abre na vertical, com um piscar no fim. */
export function panelIn(_node: Element): Css {
  if (reduced()) return { duration: 0 }
  return {
    duration: 220,
    css: (t) => {
      const s = stepped(t, 6)
      const y = Math.max(0.02, s < 0.3 ? 0.02 : (s - 0.3) / 0.7)
      const flash = s > 0.8 && s < 1 ? 'brightness(1.8) hue-rotate(40deg)' : 'none'
      return `transform: scaleY(${y}); filter: ${flash};`
    },
  }
}

export function panelOut(node: Element): Css {
  ;(node as HTMLElement).inert = true
  if (reduced()) return { duration: 0 }
  return { duration: 160, css: (t) => `transform: scaleY(${Math.max(0.02, stepped(t, 4))}); opacity: ${stepped(t, 4)};` }
}

/** Fundo dos diálogos: escurece em degraus. */
export function shade(node: Element, { out = false } = {}): Css {
  if (out) (node as HTMLElement).inert = true
  if (reduced()) return { duration: 0 }
  return { duration: 180, css: (t) => `opacity: ${stepped(t, 4)};` }
}

/** Estrela que explode em pixels a partir do elemento (favoritar). */
export function burst(from: Element | null | undefined, color = 'var(--focus)') {
  if (!from || reduced()) return
  const r = from.getBoundingClientRect()
  const cx = r.left + r.width / 2
  const cy = r.top + r.height / 2
  for (let i = 0; i < 8; i++) {
    const a = (Math.PI * 2 * i) / 8
    const px = document.createElement('span')
    px.className = 'burst-pixel'
    px.setAttribute('aria-hidden', 'true')
    px.style.left = `${cx}px`
    px.style.top = `${cy}px`
    px.style.background = color
    px.style.setProperty('--dx', `${Math.round(Math.cos(a) * 46)}px`)
    px.style.setProperty('--dy', `${Math.round(Math.sin(a) * 46)}px`)
    document.body.appendChild(px)
    setTimeout(() => px.remove(), 650)
  }
}
