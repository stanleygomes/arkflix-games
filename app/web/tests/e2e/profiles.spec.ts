import type { Page } from '@playwright/test'
import { axe, expect, open, test, type App } from './fixtures'

const call = (page: Page, app: App, m: string, args: unknown[] = []) =>
  page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })

async function typePIN(page: Page, pin: string) {
  for (const d of pin) await page.keyboard.press(d)
}

test('perfis: cria com avatar e PIN, troca, e um não vê favoritos nem saves do outro', async ({ page, app }) => {
  await open(page, app)
  // favorito e saves do perfil principal
  const smw = page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first()
  await smw.focus()
  await page.keyboard.press('f')
  await expect(smw).toHaveAttribute('aria-label', /favorito/)

  await page.getByRole('button', { name: 'Trocar perfil (Jogador 1)' }).click()
  await expect(page.getByRole('heading', { name: 'Quem vai jogar?' })).toBeVisible()
  await axe(page, 'quem vai jogar')
  await page.getByRole('button', { name: 'Adicionar perfil' }).click()
  await expect(page.getByRole('heading', { name: 'Novo perfil' })).toBeVisible()
  await page.getByRole('button', { name: /^Nome/ }).click()
  await page.getByRole('dialog', { name: 'Nome do perfil' }).getByLabel('Nome do perfil').fill('Bia')
  await page.keyboard.press('Enter')
  await page.getByRole('radio', { name: 'Gato' }).click()
  await page.getByRole('radio', { name: 'Rosa' }).click()
  await axe(page, 'editor de perfil')
  await page.getByRole('button', { name: 'Criar perfil' }).click()
  await expect(page.getByRole('heading', { name: 'Quem vai jogar?' })).toBeVisible()
  await expect(page.locator('[data-profile]')).toHaveCount(2)

  // PIN pela edição
  await page.getByRole('button', { name: 'Gerenciar perfis' }).click()
  await page.getByRole('button', { name: 'Editar Bia' }).click()
  await page.getByRole('button', { name: 'Definir PIN' }).click()
  await expect(page.getByRole('dialog', { name: 'Novo PIN' })).toBeVisible()
  await axe(page, 'teclado de PIN')
  await typePIN(page, '4321')
  await expect(page.getByRole('dialog', { name: 'Repita o novo PIN' })).toBeVisible()
  await typePIN(page, '1111')
  await expect(page.getByRole('alert')).toContainText('não batem')
  await typePIN(page, '4321')
  await expect(page.getByRole('button', { name: 'Trocar PIN' })).toBeVisible()
  const saved = app.read(`${app.data}/profiles.json`)
  expect(saved).not.toContain('4321')
  await page.getByRole('button', { name: 'Voltar' }).click()
  await page.getByRole('button', { name: 'Pronto' }).click()

  // entra na Bia: PIN errado, depois certo (no teclado da tela)
  await page.getByRole('button', { name: /^Bia/ }).click()
  await typePIN(page, '0000')
  await expect(page.getByRole('alert')).toContainText('PIN errado')
  for (const d of '4321') await page.getByRole('dialog', { name: 'PIN de Bia' }).getByRole('button', { name: d, exact: true }).click()
  await expect(page.getByRole('button', { name: 'Trocar perfil (Bia)' })).toBeVisible()
  await expect(page.locator('.shelf [data-game][aria-label*="favorito"]')).toHaveCount(0)

  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  await expect(page.getByText('Nenhum ponto salvo ainda.')).toBeVisible()
  await page.getByRole('button', { name: 'Favoritar' }).click()
  await page.getByRole('button', { name: 'Voltar' }).click()

  // volta pro principal: o favorito da Bia não aparece, o dele sim
  await page.getByRole('button', { name: 'Trocar perfil (Bia)' }).click()
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await expect(page.getByRole('button', { name: 'Trocar perfil (Jogador 1)' })).toBeVisible()
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  await expect(page.locator('.states [data-state]')).toHaveCount(2)
})

test('com mais de um perfil, o app abre em "Quem vai jogar?" e não deixa pular', async ({ page, app }) => {
  expect((await call(page, app, 'CreateProfile', ['Leo', 'robot', '#44a95a'])).ok()).toBeTruthy()
  await page.goto(app.url)
  await expect(page.getByRole('heading', { name: 'Quem vai jogar?' })).toBeVisible()
  await expect(page.locator('[data-profile]:focus')).toHaveCount(1)
  await page.keyboard.press('Escape')
  await expect(page.getByRole('heading', { name: 'Quem vai jogar?' })).toBeVisible()
  await page.keyboard.press('ArrowRight')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: 'Trocar perfil (Leo)' })).toBeVisible()
})

test('apagar perfil pede confirmação e o PIN', async ({ page, app }) => {
  const res = await (await call(page, app, 'CreateProfile', ['Tio', 'alien', '#e8902a'])).json()
  const id = res.result.id
  await call(page, app, 'SetProfilePIN', [id, '', '2468'])
  await page.goto(app.url)
  await page.getByRole('button', { name: /^Jogador 1/ }).click()
  await page.getByRole('button', { name: 'Trocar perfil (Jogador 1)' }).click()
  await page.getByRole('button', { name: 'Gerenciar perfis' }).click()
  await page.getByRole('button', { name: 'Editar Tio' }).click()
  await page.getByRole('button', { name: 'Apagar perfil' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Apagar perfil' }).click()
  await typePIN(page, '2468')
  await expect(page.getByRole('heading', { name: 'Gerenciar perfis' })).toBeVisible()
  await expect(page.locator('[data-profile]')).toHaveCount(1)
  await page.getByRole('button', { name: 'Pronto' }).click()
  await expect(page.getByRole('heading', { name: 'Quem vai jogar?' })).toBeVisible()
  expect(app.exists(`${app.data}/profiles/${id}`)).toBe(false)
})
