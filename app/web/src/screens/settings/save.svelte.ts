// Salva as configurações sem esperar a pessoa clicar em "salvar": cada mudança
// aplica na hora e grava com um pequeno atraso (várias mudanças viram uma escrita).
import { api } from '../../lib/api'
import { app, toast } from '../../lib/state.svelte'
import type { Settings } from '../../lib/types'

let timer: number | undefined

export function saveSoon(next: Settings, delay = 120) {
  app.settings = next
  clearTimeout(timer)
  timer = window.setTimeout(async () => {
    try {
      app.settings = await api.saveSettings($state.snapshot(next) as Settings)
    } catch (err) {
      toast(`Não salvou: ${(err as Error).message}`, 'error')
    }
  }, delay)
}
