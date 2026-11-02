import type { Page } from '@playwright/test'
import { axe, expect, test, type App, switchProfile } from './fixtures'

const call = async (page: Page, app: App, m: string, args: unknown[] = []) => {
  const res = await page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
  return { ok: res.ok(), body: await res.json() }
}

test('modo infantil: só jogo liberado, nada de configurar (nem por atalho) e sair pede PIN', async ({ page, app }) => {
  // o adulto libera um jogo no editor e cria o perfil infantil
  await page.goto(app.url)
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  await page.getByRole('button', { name: 'Editar' }).click()
  await page.getByRole('switch', { name: /Liberado no perfil infantil/ }).click()
  await expect(page.getByRole('switch', { name: /Liberado no perfil infantil/ })).toHaveAttribute('aria-checked', 'true')

  const nina = (await call(page, app, 'CreateProfile', ['Nina', 'mushroom', '#d8579a'])).body.result
  await call(page, app, 'SetProfilePIN', ['main', '', '9090'])
  await page.goto(app.url)
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await page.locator('.dots').waitFor()
  await page.keyboard.type('9090')
  await switchProfile(page)
  await page.getByRole('button', { name: 'Gerenciar perfis' }).click()
  await page.getByRole('button', { name: 'Editar Nina' }).click()
  await page.getByRole('switch', { name: /Perfil infantil/ }).click()
  await expect(page.getByRole('switch', { name: /Perfil infantil/ })).toHaveAttribute('aria-checked', 'true')
  await page.getByRole('button', { name: 'Voltar' }).click()
  await page.getByRole('button', { name: 'Pronto' }).click()
  await expect(page.getByText('Infantil')).toBeVisible()

  // entra como Nina
  await page.getByRole('button', { name: /^Nina/ }).click()
  await expect(page.getByRole('button', { name: /^Você: Nina/ })).toBeVisible()
  await expect(page.locator('body')).toHaveClass(/kids/)
  await expect(page.locator('.shelf [data-game]')).toHaveCount(1)
  await axe(page, 'início infantil')

  // Start abre a tela Você só com o que é dela: nada do console
  await page.keyboard.press('m')
  await expect(page.locator('main.you')).toBeVisible()
  await expect(page.getByRole('heading', { name: 'Deste console' })).toHaveCount(0)
  await expect(page.locator('main.you [data-item]')).toHaveText([/Central/, /Estatísticas/])
  await axe(page, 'você infantil')
  await page.keyboard.press('Escape')
  await page.goto(`${app.url}/`)
  await expect(page.locator('.shelf [data-game]')).toHaveCount(1)

  // na página do jogo: sem editar, e o botão de cima não apaga save
  await page.locator('.shelf [data-game]').first().click()
  await expect(page.getByRole('button', { name: 'Editar' })).toHaveCount(0)

  // a API também trava
  const s = (await call(page, app, 'GetSettings')).body.result
  const saved = await call(page, app, 'SaveSettings', [s])
  expect(saved.ok).toBe(false)
  expect(JSON.stringify(saved.body)).toContain('bloqueado no perfil infantil')
  const all = (await call(page, app, 'Bootstrap')).body.result.games
  expect(all).toHaveLength(1)
  expect((await call(page, app, 'SetProfileKids', [nina.id, false])).ok).toBe(false)

  // sair pro adulto pede o PIN dele
  await switchProfile(page)
  await expect(page.getByRole('button', { name: 'Gerenciar perfis' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Adicionar perfil' })).toHaveCount(0)
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await expect(page.getByRole('dialog', { name: 'PIN de Jogador 1' })).toBeVisible()
  await page.keyboard.type('1111')
  await expect(page.getByRole('alert')).toContainText('PIN errado')
  await page.keyboard.type('9090')
  await expect(page.getByRole('button', { name: 'Configurações' })).toBeVisible()
  await expect(page.locator('body')).not.toHaveClass(/kids/)
})
