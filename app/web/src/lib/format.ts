// Textos curtos pra tela: tempo de jogo, datas, tamanhos.

export function playTime(seconds: number): string {
  if (!seconds || seconds < 60) return seconds > 0 ? 'menos de 1 min' : 'nunca jogado'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h === 0) return `${m} min`
  return m ? `${h} h ${m} min` : `${h} h`
}

export function ago(unix: number, now = Date.now() / 1000): string {
  if (!unix) return 'nunca'
  const diff = Math.max(0, now - unix)
  if (diff < 60) return 'agora'
  if (diff < 3600) return `há ${Math.floor(diff / 60)} min`
  if (diff < 86400) return `há ${Math.floor(diff / 3600)} h`
  const days = Math.floor(diff / 86400)
  if (days === 1) return 'ontem'
  if (days < 30) return `há ${days} dias`
  return new Date(unix * 1000).toLocaleDateString('pt-BR')
}

export function bytes(n: number): string {
  if (n < 1024) return `${n} B`
  const units = ['KB', 'MB', 'GB', 'TB']
  let v = n / 1024
  let i = 0
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(v < 10 ? 1 : 0).replace('.', ',')} ${units[i]}`
}

export function dateTime(unix: number): string {
  return new Date(unix * 1000).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' })
}
