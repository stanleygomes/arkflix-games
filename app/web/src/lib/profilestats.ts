// Resumo das estatísticas pro cartão do perfil (tela Você). Puro.
import type { Stats } from './types'

export interface ProfileSummary {
  weekSeconds: number
  week: { label: string; date: string; seconds: number }[] // 7 dias, hoje por último
  streak: number // dias seguidos jogando (até hoje ou ontem)
  topGame: string
  topSystem: string
}

const LETTERS = ['D', 'S', 'T', 'Q', 'Q', 'S', 'S']

function iso(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

export function summarize(s: Stats, today = new Date()): ProfileSummary {
  const by = new Map(s.byDay.map((d) => [d.date, d.seconds]))
  const week = []
  for (let i = 6; i >= 0; i--) {
    const d = new Date(today.getFullYear(), today.getMonth(), today.getDate() - i)
    week.push({ label: LETTERS[d.getDay()], date: iso(d), seconds: by.get(iso(d)) ?? 0 })
  }
  // sequência: conta pra trás a partir de hoje (ou de ontem, se hoje ainda não jogou)
  let streak = 0
  let day = new Date(today.getFullYear(), today.getMonth(), today.getDate())
  if (!(by.get(iso(day)) ?? 0)) day = new Date(day.getFullYear(), day.getMonth(), day.getDate() - 1)
  while ((by.get(iso(day)) ?? 0) > 0) {
    streak++
    day = new Date(day.getFullYear(), day.getMonth(), day.getDate() - 1)
  }
  const top = [...s.byGame].sort((a, b) => b.seconds - a.seconds)[0]
  const sys = [...s.bySystem].filter((x) => x.seconds > 0).sort((a, b) => b.seconds - a.seconds)[0]
  return {
    weekSeconds: week.reduce((t, d) => t + d.seconds, 0),
    week,
    streak,
    topGame: top && top.seconds > 0 ? top.name : '',
    topSystem: sys?.system ?? '',
  }
}
