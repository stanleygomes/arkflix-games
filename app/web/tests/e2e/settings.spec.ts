import { axe, expect, open, test, settings, pickItem } from './fixtures'

test('acessibilidade: fonte, alto contraste e animações aplicam na hora e persistem', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await pickItem(page, 'Acessibilidade e som')
  await axe(page, 'configurações: acessibilidade')

  await page.getByRole('button', { name: '150%' }).click()
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).fontSize)).toBe('24px')

  await page.getByRole('switch', { name: 'Alto contraste' }).click()
  await expect(page.locator('body')).toHaveClass(/high-contrast/)
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--bg').trim())).toBe('rgb(0, 0, 0)')
  await axe(page, 'alto contraste')

  await page.getByRole('switch', { name: 'Reduzir animações' }).click()
  await expect(page.locator('body')).toHaveClass(/reduce-motion/)
  await page.getByRole('button', { name: 'Diminuir volume da música' }).click()
  await expect(page.getByRole('meter', { name: 'Volume da música' })).toHaveAttribute('aria-valuenow', '15')

  await expect.poll(() => JSON.parse(app.read(`${app.data}/settings.json`)).a11y.highContrast).toBe(true)
  await page.reload()
  await expect(page.locator('body')).toHaveClass(/high-contrast/)
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).fontSize)).toBe('24px')
})

test('títulos sem fonte pixel trocam a fonte de verdade', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await pickItem(page, 'Acessibilidade e som')
  await page.getByRole('switch', { name: 'Títulos em fonte pixel' }).click()
  await expect(page.locator('body')).toHaveClass(/no-pixel/)
  const font = await page.locator('h1').first().evaluate((el) => getComputedStyle(el).fontFamily)
  expect(font).toContain('Atkinson')
})

test('biblioteca: adicionar pasta e salvar procura os jogos de novo', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await expect(page.getByRole('heading', { name: 'Biblioteca', level: 1 })).toBeVisible()
  await axe(page, 'configurações: biblioteca')
  const save = page.getByRole('button', { name: 'Salvar' })
  await expect(save).toBeDisabled()
  await page.getByRole('button', { name: 'Remover pasta' }).last().click()
  await save.click()
  await expect(page.locator('.toasts')).toContainText('6 jogos na biblioteca')
  const saved = JSON.parse(app.read(`${app.data}/settings.json`))
  expect(saved.libraryDirs).toHaveLength(1)
})

test('sobre: versão e atalhos', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await page.keyboard.press('Escape') // do painel volta pra lista
  // R1 anda pela lista: biblioteca → importar, servidor, armazenamento, emuladores, bluetooth, atualizações, perfis, protetor, sobre
  for (let i = 0; i < 9; i++) await page.keyboard.press('e')
  await expect(page.getByRole('heading', { name: 'Sobre' })).toBeVisible()
  await expect(page.locator('.facts').first()).toContainText('Navegador')
  await expect(page.getByRole('definition').filter({ hasText: '10-registra.sh' })).toBeVisible()
  await axe(page, 'configurações: sobre')
})
