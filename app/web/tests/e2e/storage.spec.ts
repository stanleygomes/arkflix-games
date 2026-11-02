import { axe, expect, open, test } from './fixtures'

test('armazenamento: disco, jogos por console e saves', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('m')
  await page.getByRole('button', { name: 'Armazenamento', exact: true }).click()
  await expect(page.getByRole('heading', { name: /livres de/ })).toBeVisible()
  const bar = page.getByRole('img', { name: /livres de .*Jogos: .*Saves: / })
  await expect(bar).toBeVisible()
  for (const label of ['Jogos', 'Saves', 'App e capas', 'Outros programas e arquivos', 'Livre']) {
    await expect(page.locator('.legend').getByText(label, { exact: true })).toBeVisible()
  }
  await expect(page.getByRole('heading', { name: /Jogos por console · 9 jogos/ })).toBeVisible()
  await expect(page.locator('.rows').first()).toContainText('Super Nintendo')
  await expect(page.getByRole('heading', { name: /^Saves · / })).toBeVisible()
  await expect(page.getByText('RetroArch (save states)')).toBeVisible()
  await axe(page, 'armazenamento')
})
