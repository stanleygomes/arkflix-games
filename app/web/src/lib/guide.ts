// Guia rápido (botão Home): abre por cima de qualquer tela. Com jogo aberto,
// traz o app pra frente do emulador; "Voltar ao jogo" devolve.
import { api } from './api'
import { play } from './sound'
import { app } from './state.svelte'

export function openGuide() {
  play('accept')
  app.guide = true
  if (app.playing?.state === 'running') api.showGuide().catch(() => {})
}

export function closeGuide() {
  app.guide = false
  if (app.playing?.state === 'running') api.resumeGame().catch(() => {})
}
