// Glitch analógico (sinal de TV, nunca digital).
// - Por evento: erro = estática; trocar de perfil = desmagnetizar; jogo
//   fechado = rolagem vertical; troca de tela = chuvisco.
// - Ambiente (startAmbient): de tempos em tempos um título pisca como letreiro,
//   separa as cores ou "rasga"; mais raro, a tela inteira treme ou chuvisca.
//   Só em títulos (h1/h2/h3, rótulos pixel): texto corrido fica limpo.
// Some com "reduzir animações" e no alto contraste.

export type Glitch = 'static' | 'jitter' | 'degauss' | 'hold' | 'snow' | 'bar'

const MS: Record<Glitch, number> = { static: 260, jitter: 260, degauss: 1000, hold: 700, snow: 220, bar: 900 }

let timer = 0

export function quiet(): boolean {
  const body = document.body.classList
  return (
    body.contains('reduce-motion') ||
    body.contains('high-contrast') ||
    matchMedia('(prefers-reduced-motion: reduce)').matches
  )
}

/** Dispara o glitch na tela inteira; um por vez, o novo substitui o anterior. */
export function glitch(kind: Glitch): void {
  if (quiet()) return
  const root = document.documentElement
  clearTimeout(timer)
  delete root.dataset.glitch
  void root.offsetWidth // reinicia a animação se for o mesmo
  root.dataset.glitch = kind
  timer = window.setTimeout(() => delete root.dataset.glitch, MS[kind])
}

// ---- glitch nos títulos (Web Animations: soma às animações do CSS, não troca)

const TITLES = '.layer :is(h1, h2, h3, .pixel, .sub, .logo)'

type TextGlitch = (el: HTMLElement) => Animation

const RED = 'rgb(255 0 77 / 0.85)'
const CYAN = 'rgb(41 173 255 / 0.85)'

const TEXT: TextGlitch[] = [
  // letreiro com mau contato: apaga e acende em degraus
  (el) =>
    el.animate(
      [{ opacity: 1 }, { opacity: 0.25 }, { opacity: 1 }, { opacity: 0.4 }, { opacity: 1 }, { opacity: 0.15 }, { opacity: 1 }],
      { duration: 520, easing: 'steps(7, end)' },
    ),
  // cores do tubo se separam e tremem
  (el) =>
    el.animate(
      [
        { textShadow: `-3px 0 ${RED}, 3px 0 ${CYAN}`, transform: 'translateX(1px)' },
        { textShadow: `2px 0 ${RED}, -2px 0 ${CYAN}`, transform: 'translateX(-2px)' },
        { textShadow: `-1px 0 ${RED}, 4px 0 ${CYAN}`, transform: 'translateX(1px)' },
        { textShadow: 'none', transform: 'none' },
      ],
      { duration: 300, easing: 'steps(4, end)' },
    ),
  // rasgo: uma faixa da palavra escorrega pro lado
  (el) =>
    el.animate(
      [
        { clipPath: 'inset(0 0 55% 0)', transform: 'translateX(6px)' },
        { clipPath: 'inset(40% 0 20% 0)', transform: 'translateX(-5px)' },
        { clipPath: 'inset(70% 0 0 0)', transform: 'translateX(3px)' },
        { clipPath: 'inset(0)', transform: 'none' },
      ],
      { duration: 240, easing: 'steps(4, end)' },
    ),
]

function visibleTitles(): HTMLElement[] {
  const h = window.innerHeight
  return [...document.querySelectorAll<HTMLElement>(TITLES)].filter((el) => {
    const r = el.getBoundingClientRect()
    return r.width > 0 && r.height > 0 && r.bottom > 0 && r.top < h && !el.closest('[role="dialog"]')
  })
}

const pick = <T>(xs: T[]): T => xs[Math.floor(Math.random() * xs.length)]
const between = (a: number, b: number) => a + Math.random() * (b - a)

/** Glitch num título visível qualquer. */
export function glitchText(): void {
  if (quiet()) return
  const titles = visibleTitles()
  if (titles.length) pick(TEXT)(pick(titles))
}

/** Liga os glitches de ambiente. `paused()` = não agora (protetor, jogo aberto).
 * Devolve a função que desliga. */
export function startAmbient(paused: () => boolean): () => void {
  let alive = true
  let t1 = 0
  let t2 = 0
  const busy = () => !alive || document.hidden || paused() || quiet()
  // títulos: a cada 4–9 s
  const text = () => {
    if (!busy()) glitchText()
    t1 = window.setTimeout(text, between(4000, 9000))
  }
  // tela inteira, sutil: a cada 12–20 s
  const screen = () => {
    if (!busy()) glitch(pick<Glitch>(['jitter', 'snow', 'bar', 'snow']))
    t2 = window.setTimeout(screen, between(12000, 20000))
  }
  t1 = window.setTimeout(text, between(2500, 5000))
  t2 = window.setTimeout(screen, between(12000, 20000))
  return () => {
    alive = false
    clearTimeout(t1)
    clearTimeout(t2)
  }
}
