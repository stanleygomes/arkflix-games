import { axe, expect, open, test, settings } from './fixtures'

test('servidor RomM: entra, lista, baixa um jogo e sai', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await page.getByRole('button', { name: 'Servidor', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Servidor RomM' })).toBeVisible()
  await axe(page, 'servidor desconectado')

  await page.getByRole('button', { name: /^Endereço/ }).click()
  await page.getByRole('dialog', { name: 'Endereço do servidor' }).getByLabel('Endereço do servidor').fill(`${app.url}/fake-romm`)
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: /^Usuário/ }).click()
  await page.getByRole('dialog', { name: 'Usuário do RomM' }).getByLabel('Usuário do RomM').fill('stanley')
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: 'Entrar' }).click()

  const pass = page.getByRole('dialog', { name: 'Senha do RomM' }).getByLabel('Senha do RomM')
  await expect(pass).toHaveAttribute('type', 'password')
  await pass.fill('errada')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('alert')).toContainText('senha')
  await pass.fill('senha123')
  await page.keyboard.press('Enter')
  await expect(page.locator('.panel-body .status')).toContainText('Conectado a')
  await expect(page.getByText(/Jogos do servidor: 3 jogos no servidor/)).toBeVisible()
  await expect(page.getByText(/3 jogos no servidor/).first()).toBeVisible()
  const saved = app.read(`${app.data}/romm.json`)
  expect(saved).not.toContain('senha123')
  await axe(page, 'servidor conectado')

  // na biblioteca: jogo remoto com selo, coleção "Servidor"
  await page.keyboard.press('Escape') // volta pra tela Você
  await page.keyboard.press('Escape') // e pro Início
  const dkc = page.locator('.shelf [data-game][aria-label^="Donkey Kong Country"]').first()
  await expect(dkc).toHaveAttribute('aria-label', /no servidor/)
  const server = page.locator('.shelf', { has: page.locator('h2', { hasText: 'No servidor' }) })
  await expect(server.locator('[data-game]')).toHaveCount(2) // DKC e Spyro (SMW já está aqui)

  await dkc.click()
  await expect(page.locator('.facts')).toContainText('No servidor')
  await expect(page.getByRole('button', { name: 'Jogar do início' })).toHaveCount(0)
  await axe(page, 'jogo no servidor')
  await page.getByRole('button', { name: /Baixar do servidor/ }).click()
  await expect(page.getByRole('button', { name: 'Jogar do início' })).toBeVisible({ timeout: 10_000 })
  await expect(page.locator('.facts')).not.toContainText('No servidor')
  expect(app.exists('home/Games/roms/snes/Donkey Kong Country (U).zip')).toBe(true)

  // sair do servidor tira os remotos (o baixado fica)
  await page.keyboard.press('Escape')
  await settings(page)
  await page.getByRole('button', { name: 'Servidor', exact: true }).click()
  await page.getByRole('button', { name: 'Sair do servidor' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Sair do servidor' }).click()
  await expect(page.getByRole('button', { name: 'Entrar' })).toBeVisible()
  expect(app.exists(`${app.data}/romm.json`)).toBe(false)
  await page.keyboard.press('Escape') // Você
  await page.keyboard.press('Escape') // Início
  await expect(page.locator('.shelf [data-game][aria-label*="no servidor"]')).toHaveCount(0)
  await expect(page.locator('.shelf [data-game][aria-label^="Donkey Kong Country"]').first()).toBeVisible()
})
