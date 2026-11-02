import { axe, expect, goTab, open, test, settings } from './fixtures'

test('capas do libretro: liga, baixa com progresso e aparece na biblioteca', async ({ page, app }) => {
  await open(page, app)
  await goTab(page, 'snes')
  await expect(page.locator('.shelf [data-game][aria-label^="Mega Man X"] img')).toHaveCount(0)

  await settings(page)
  await expect(page.getByRole('button', { name: 'Biblioteca', exact: true })).toBeFocused()
  await expect(page.getByRole('button', { name: 'Atualizar capas agora' })).toBeDisabled()
  await page.getByRole('switch', { name: /Baixar capas automaticamente/ }).click()
  await expect(page.locator('.toasts')).toContainText('Baixando capas:', { timeout: 15000 })
  await expect(page.locator('.toasts')).toContainText('com capa')
  await axe(page, 'configurações com capas ligadas')
  expect(JSON.parse(app.read(`${app.data}/settings.json`)).covers.enabled).toBe(true)

  await page.keyboard.press('Escape') // volta pra tela Você
  await page.keyboard.press('Escape') // e pro Início
  await expect(page.locator('.shelf [data-game][aria-label^="Mega Man X"] img')).toHaveCount(1)
  const width = await page.locator('.shelf [data-game][aria-label^="Mega Man X"] img').evaluate(async (img: HTMLImageElement) => {
    if (!img.complete) await new Promise((r) => (img.onload = r))
    return img.naturalWidth
  })
  expect(width).toBeGreaterThan(0)

  // de novo: nada a baixar
  await settings(page)
  await page.getByRole('button', { name: 'Atualizar capas agora' }).click()
  await expect(page.locator('.toasts')).toContainText('em dia')
})
