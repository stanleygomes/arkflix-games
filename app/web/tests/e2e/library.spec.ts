import { axe, expect, goTab, open, test } from './fixtures'

test('ver tudo abre a grade do console; ordem escolhida fica salva e vale na fileira', async ({ page, app }) => {
  await open(page, app)
  await goTab(page, 'snes')
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

test('filtros combinam, contam e explicam lista vazia', async ({ page, app }) => {
  await open(page, app)
  await goTab(page, 'snes')
  await page.getByRole('button', { name: 'Ver tudo' }).click()
  await page.keyboard.press('m')
  const dialog = page.getByRole('dialog', { name: 'Opções da lista' })
  await dialog.getByRole('checkbox', { name: 'Com save' }).click()
  await expect(dialog.getByRole('checkbox', { name: 'Com save' })).toHaveAttribute('aria-checked', 'true')
  await axe(page, 'opções com filtro')
  await page.keyboard.press('Escape')
  await expect(page.locator('.grid .name')).toHaveText(['Super Mario World'])
  await expect(page.locator('.head')).toContainText('1 jogo de 2')
  await expect(page.locator('.head')).toContainText('1 filtro ativo')

  await page.keyboard.press('m')
  await dialog.getByRole('checkbox', { name: 'Ação' }).click() // Mega Man X: Ação, sem save
  await page.keyboard.press('Escape')
  await expect(page.locator('.body')).toContainText('Nenhum jogo com esses filtros')
  await page.getByRole('button', { name: 'Limpar filtros' }).click()
  await expect(page.locator('.grid .name')).toHaveCount(2)
})

test('surpresa sorteia dentro do console e não repete o último', async ({ page, app }) => {
  await open(page, app)
  await goTab(page, 'snes')
  await page.keyboard.press('ArrowDown')
  const snes = ['Mega Man X', 'Super Mario World']
  let last = ''
  for (let i = 0; i < 4; i++) {
    await page.keyboard.press('r')
    await expect(page.locator('.toasts')).toContainText('Surpresa:')
    await expect(page.locator('.shelf [data-game]:focus')).toHaveCount(1, { timeout: 4000 })
    const title = (await page.locator('#hero-title').textContent())!.trim()
    expect(snes).toContain(title)
    expect(title).not.toBe(last) // com 2 jogos, alterna
    last = title
  }
  await page.getByRole('button', { name: 'Ver tudo' }).click()
  await app.pad('r3')
  await expect(page.locator('.grid [data-game]:focus')).toHaveCount(1)
})
