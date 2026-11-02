import { writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { axe, expect, open, test, openItem } from './fixtures'

test('estatísticas: totais, rankings e períodos a partir do histórico', async ({ page, app }) => {
  // histórico conhecido: 2 partidas de SMW (1 h + 30 min) hoje/ontem, 1 de Sonic há 10 dias, 1 antiga
  const ids: Record<string, string> = await page.request
    .post(`${app.url}/api/call/Bootstrap`, { headers: { 'X-Arkflix': '1' }, data: [] })
    .then(async (r) => Object.fromEntries(((await r.json()).result.games as { id: string; name: string }[]).map((g) => [g.name, g.id])))
  const now = Math.floor(Date.now() / 1000)
  const day = 86400
  const lines = [
    { game: ids['Super Mario World'], system: 'snes', start: now - 3600, seconds: 3600, mode: 'start', code: 0 },
    { game: ids['Super Mario World'], system: 'snes', start: now - day, seconds: 1800, mode: 'auto', code: 0 },
    { game: ids['Sonic The Hedgehog 2'], system: 'genesis', start: now - 10 * day, seconds: 600, mode: 'start', code: 0 },
    { game: ids['Super Mario 64'], system: 'n64', start: now - 60 * day, seconds: 7200, mode: 'start', code: 0 },
  ]
  writeFileSync(join(app.root, app.data, 'sessions.jsonl'), lines.map((l) => JSON.stringify(l)).join('\n') + '\n')

  await open(page, app)
  await openItem(page, 'Estatísticas')
  await expect(page.getByRole('heading', { name: 'Estatísticas', level: 1 })).toBeVisible()
  const tiles = page.locator('.tiles')
  await expect(tiles).toContainText('3 h 40 min')
  await expect(tiles).toContainText('Partidas4')
  await expect(tiles).toContainText('Jogos jogados3 de 9')
  await expect(page.locator('.chart').first().locator('.name')).toHaveText(['Super Mario 64', 'Super Mario World', 'Sonic The Hedgehog 2'])
  await axe(page, 'estatísticas')

  await page.getByRole('button', { name: '30 dias' }).click()
  await expect(tiles).toContainText('1 h 40 min')
  await expect(tiles).toContainText('Jogos jogados2 de 9')

  await page.getByRole('button', { name: '7 dias' }).click()
  await expect(tiles).toContainText('1 h 30 min')
  // tabela no lugar do gráfico
  await page.getByRole('button', { name: 'Ver tabela' }).first().click()
  await expect(page.getByRole('table').first()).toContainText('Super Mario World')
  await axe(page, 'estatísticas em tabela')
})

test('sem partidas: explica em vez de gráfico vazio', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Estatísticas')
  await expect(page.locator('.empty')).toContainText('Nenhuma partida')
})
