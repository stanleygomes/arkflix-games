import type { Page } from '@playwright/test'
import { writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { axe, expect, test, type App, switchProfile } from './fixtures'

const call = async (page: Page, app: App, m: string, args: unknown[] = []) => {
  const res = await page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
  return { ok: res.ok(), body: await res.json() }
}

// histórico do perfil: sessões que terminam agora, somando `minutes` hoje
const played = (app: App, id: string, game: string, minutes: number) => {
  const now = Math.floor(Date.now() / 1000)
  const line = { game, system: 'snes', name: 'Super Mario World', start: now - minutes * 60, seconds: minutes * 60, mode: 'start', code: 0 }
  writeFileSync(join(app.root, app.data, 'profiles', id, 'sessions.jsonl'), JSON.stringify(line) + '\n')
}

test('limite do perfil infantil: o adulto define, a criança vê o que resta e, zerado, não joga', async ({ page, app }) => {
  test.slow() // três varreduras de acessibilidade, uma na grade de avatares
  await page.goto(app.url)
  const games = (await call(page, app, 'Bootstrap')).body.result.games as { id: string; name: string }[]
  const smw = games.find((g) => g.name === 'Super Mario World')!
  await call(page, app, 'SaveMeta', [smw.id, { kidsOk: true, rating: 0, completed: false, broken: false }])
  const nina = (await call(page, app, 'CreateProfile', ['Nina', 'mushroom', '#d8579a'])).body.result

  // o adulto liga o infantil e escolhe 1 h por dia (as opções só aparecem com o infantil ligado)
  await page.goto(app.url)
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await switchProfile(page)
  await page.getByRole('button', { name: 'Gerenciar perfis' }).click()
  await page.getByRole('button', { name: 'Editar Nina' }).click()
  await expect(page.getByRole('group', { name: 'Tempo por dia' })).toHaveCount(0)
  await page.getByRole('switch', { name: /Perfil infantil/ }).click()
  const daily = page.getByRole('group', { name: 'Tempo por dia' })
  await expect(daily.getByRole('button', { name: 'Sem limite' })).toHaveAttribute('aria-pressed', 'true')
  await daily.getByRole('button', { name: '1 h' }).click()
  await expect(daily.getByRole('button', { name: '1 h' })).toHaveAttribute('aria-pressed', 'true')
  await expect(page.getByRole('group', { name: 'Hora de dormir' }).getByRole('button', { name: 'Sem' })).toHaveAttribute('aria-pressed', 'true')
  await axe(page, 'editor de perfil com limite')
  await page.getByRole('button', { name: 'Voltar' }).click()
  await page.getByRole('button', { name: 'Pronto' }).click()
  await expect(page.getByRole('button', { name: 'Gerenciar perfis' })).toBeVisible()

  // a criança jogou 35 min hoje: restam 25
  played(app, nina.id, smw.id, 35)
  await call(page, app, 'SelectProfile', [nina.id, ''])
  await page.goto(app.url)
  await expect(page.getByRole('button', { name: /^Você: Nina/ })).toBeVisible()
  await page.keyboard.press('m')
  await expect(page.locator('main.you')).toContainText('Hoje: restam 25 min')
  await axe(page, 'você com limite')
  // e não muda o próprio limite
  expect((await call(page, app, 'SetProfileLimits', [nina.id, 0, ''])).ok).toBe(false)

  // acabou o tempo: Jogar desligado com o motivo, e a API recusa
  played(app, nina.id, smw.id, 61)
  await page.goto(app.url)
  await page.locator('.shelf [data-game]').first().click()
  await expect(page.getByRole('button', { name: 'Jogar do início' })).toBeDisabled()
  await expect(page.getByRole('status').filter({ hasText: 'Acabou o tempo de hoje. Volta amanhã!' })).toBeVisible()
  await axe(page, 'jogo sem tempo')
  const res = await call(page, app, 'Play', [smw.id, ''])
  expect(res.ok).toBe(false)
  expect(JSON.stringify(res.body)).toContain('Acabou o tempo de hoje')
  expect(app.calls()).not.toContain('Super Mario World')
})
