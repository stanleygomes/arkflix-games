import { axe, expect, open, test } from './fixtures'

test('visualizador de mídia: capa, print, saves e manual, no teclado', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').click()
  const strip = page.locator('.strip button')
  await expect(strip).toHaveCount(5) // capa, tela do jogo, 2 saves, manual
  await expect(strip.last()).toContainText('Manual')

  await page.getByRole('button', { name: 'Ver Tela do jogo' }).click()
  const viewer = page.getByRole('dialog', { name: 'Tela do jogo' })
  await expect(viewer).toBeVisible()
  await expect(viewer.locator('.counter')).toHaveText('2 / 5')
  const img = viewer.locator('img')
  await expect.poll(() => img.evaluate((i) => (i as HTMLImageElement).naturalWidth)).toBeGreaterThan(0)
  await axe(page, 'visualizador de mídia')

  await page.keyboard.press('Enter')
  await expect(viewer.getByRole('button', { name: 'Zoom' })).toHaveAttribute('aria-pressed', 'true')
  await page.keyboard.press('ArrowLeft')
  await expect(page.getByRole('dialog', { name: 'Capa' })).toBeVisible()
  await page.keyboard.press('ArrowLeft') // dá a volta
  const manual = page.getByRole('dialog', { name: 'Manual' })
  await expect(manual).toBeVisible()
  await expect(manual.locator('iframe')).toHaveAttribute('src', /\/media\/manual\//)
  const src = (await manual.locator('iframe').getAttribute('src')) as string
  const pdf = await page.request.get(new URL(src, page.url()).href)
  expect((await pdf.body()).subarray(0, 4).toString()).toBe('%PDF')

  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.locator('h1')).toHaveText('Super Mario World')
})

test('jogo sem mídia não mostra a faixa', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Mega Man X"]').first().click()
  await expect(page.locator('h1')).toHaveText('Mega Man X')
  await expect(page.locator('.strip')).toHaveCount(0)
})
