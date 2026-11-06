// Glitch analógico (sinal de TV, nunca digital), sempre ligado a um evento:
// erro = estática + tremida; trocar de perfil = desmagnetizar; jogo fechado =
// rolagem vertical. Some com "reduzir animações" e no alto contraste.

export type Glitch = 'static' | 'jitter' | 'degauss' | 'hold'

const MS: Record<Glitch, number> = { static: 260, jitter: 260, degauss: 1000, hold: 700 }

let timer = 0

function quiet(): boolean {
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
