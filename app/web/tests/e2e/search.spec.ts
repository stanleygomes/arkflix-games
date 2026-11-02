import { axe, expect, open, test } from './fixtures'

test('busca pelo teclado físico, sem acento nem maiúscula', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('/')
  await expect(page.getByRole('heading', { name: 'Buscar' })).toBeVisible()
  await expect(page.getByLabel(/Buscar jogo/)).toBeFocused()
  await page.keyboard.type('MARIO')
  await expect(page.locator('#results-count')).toHaveText('3 jogos')
  await axe(page, 'busca')
  await page.keyboard.press('Enter') // vai pros resultados
  await expect(page.locator('.results [data-game]').first()).toBeFocused()
  await page.keyboard.press('Enter')
  await expect(page.locator('h1')).toHaveText(/Mario/)
})

test('busca só com o controle (teclado na tela)', async ({ page, app }) => {
  await open(page, app)
  await app.pad('select')
  await expect(page.getByRole('heading', { name: 'Buscar' })).toBeVisible()
  for (const key of ['s', 'o', 'n']) {
    await page.locator(`.osk [data-key="${key}"]`).focus()
    await app.pad('south')
  }
  await expect(page.getByLabel(/Buscar jogo/)).toHaveValue('son')
  await expect(page.locator('.results [data-game]')).toHaveCount(1)
  await page.getByRole('button', { name: 'Apagar' }).focus()
  await app.pad('south')
  await expect(page.getByLabel(/Buscar jogo/)).toHaveValue('so')
  await page.getByRole('button', { name: 'Ver resultados' }).focus()
  await app.pad('south')
  await expect(page.locator('.results [data-game]').first()).toBeFocused()
})

test('nada encontrado explica', async ({ page, app }) => {
  await open(page, app)
  await page.getByRole('button', { name: 'Buscar jogos' }).click()
  await page.getByLabel(/Buscar jogo/).fill('zelda')
  await expect(page.locator('#results-count')).toHaveText('Nada encontrado pra "zelda".')
})

test('busca global: console, configuração e energia', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('/')
  const input = page.getByLabel(/Buscar jogo/)
  await input.fill('blueto')
  await expect(page.locator('[data-shortcut="setting"]')).toHaveText([/Bluetooth/])
  await axe(page, 'busca com atalhos')
  await page.locator('[data-shortcut="setting"]').click()
  await expect(page.getByRole('heading', { name: 'Bluetooth' })).toBeVisible()
  await page.keyboard.press('Escape')
  await input.fill('super nin')
  await page.locator('[data-shortcut="console"]', { hasText: 'Super Nintendo' }).click()
  await expect(page.getByRole('heading', { name: 'Super Nintendo' })).toBeVisible()
})
