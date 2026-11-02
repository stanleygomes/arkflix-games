import { axe, expect, open, test } from './fixtures'

test('guia rápido: abre com G, atalhos e central com avisos', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('f') // favorita: gera um aviso
  await expect(page.getByRole('button', { name: /aviso novo/ })).toBeVisible()
  await page.keyboard.press('g')
  const guide = page.getByRole('dialog', { name: 'Jogador 1' })
  await expect(guide).toBeVisible()
  await expect(guide.getByRole('button', { name: /Início/ })).toBeFocused()
  await expect(guide.getByRole('button', { name: /Central/ })).toContainText('1 novo')
  await axe(page, 'guia rápido')
  await page.keyboard.press('Escape')
  await expect(guide).toBeHidden()

  await page.keyboard.press('g')
  await guide.getByRole('button', { name: /Central/ }).click()
  await expect(page.getByRole('heading', { name: 'Central' })).toBeVisible()
  await expect(page.locator('.notice')).toContainText('favoritos')
  await axe(page, 'central')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('button', { name: /aviso novo/ })).toHaveCount(0) // visto
})

test.describe('com jogo aberto', () => {
  test.use({ fakeSleep: '30' })
  test('guia por cima do jogo: fecha o jogo', async ({ page, app }) => {
    await open(page, app)
    await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
    await page.keyboard.press('g')
    const guide = page.getByRole('dialog', { name: 'Jogador 1' })
    await expect(guide.getByRole('button', { name: 'Voltar ao jogo' })).toBeFocused()
    await expect(guide.getByRole('button', { name: 'Salvar estado' })).toBeVisible()
    await axe(page, 'guia com jogo')
    await guide.getByRole('button', { name: 'Fechar jogo' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Fechar jogo' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
    await expect(page.locator('.toasts')).toContainText('Super Mario World')
  })
})
