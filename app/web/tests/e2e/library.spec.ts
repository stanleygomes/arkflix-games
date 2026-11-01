import { axe, expect, open, test } from './fixtures'

test('ver tudo abre a grade do console; ordem escolhida fica salva e vale na fileira', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('ArrowRight')
  await page.keyboard.press('ArrowRight') // SNES
  await page.getByRole('button', { name: 'Ver tudo' }).click()
  await expect(page.getByRole('heading', { name: 'Super Nintendo' })).toBeVisible()
  const names = page.locator('.grid .name')
  await expect(names).toHaveText(['Mega Man X', 'Super Mario World'])
  await axe(page, 'biblioteca em grade')

  await page.keyboard.press('m') // Start / opções
  const dialog = page.getByRole('dialog', { name: 'Opções da lista' })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('radio', { name: 'Nome (A–Z)' })).toBeFocused()
  await axe(page, 'opções da lista')
  await dialog.getByRole('radio', { name: 'Ano (mais antigos)' }).click()
  await expect(dialog.getByRole('radio', { name: 'Ano (mais antigos)' })).toHaveAttribute('aria-checked', 'true')
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(names).toHaveText(['Super Mario World', 'Mega Man X']) // 1990, 1993
  const settings = `${app.data}/settings.json`
  await expect.poll(() => (app.exists(settings) ? JSON.parse(app.read(settings)).ui.sort['system:snes'] : null)).toBe('yearAsc')

  await page.keyboard.press('Escape') // volta pro início, ainda no SNES
  await expect(page.locator('.shelf [data-game]').first()).toHaveAttribute('aria-label', /^Super Mario World/)
})
