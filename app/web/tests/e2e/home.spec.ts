import { axe, expect, focused, open, test } from './fixtures'

test('tela inicial: consoles com jogos, destaque e acessibilidade', async ({ page, app }) => {
  await open(page, app)
  const tabs = page.locator('nav.tabs [data-tab]')
  await expect(tabs).toHaveText(['Início', 'NES', 'SNES', 'MD', 'N64', 'PS1', 'PS2', 'PSP', 'ARC'].map((t) => new RegExp(`^${t}`)))
  await expect(page.locator('nav.tabs [aria-current="page"]')).toHaveText(/^Início/)
  await expect(page.locator('#hero-title')).toBeVisible()
  await expect(page.locator('.shelf [data-game]')).toHaveCount(9)
  await axe(page, 'tela inicial')
})

test('teclado: troca console, desce pra fileira e o destaque acompanha', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('ArrowRight')
  await page.keyboard.press('ArrowRight')
  await expect(page.locator('nav.tabs [aria-current="page"]')).toHaveText(/^SNES/)
  const accent = await page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--accent').trim())
  expect(accent).toBe('#a99bff') // paleta do SNES

  await page.keyboard.press('ArrowDown')
  expect((await focused(page)).label).toContain('Mega Man X')
  await expect(page.locator('#hero-title')).toHaveText('Mega Man X')
  await expect(page.locator('.hero .summary')).toContainText('Sigma')

  await page.keyboard.press('ArrowRight')
  await expect(page.locator('#hero-title')).toHaveText('Super Mario World')
  await page.keyboard.press('ArrowUp')
  expect((await focused(page)).text).toContain('Jogar')
  await page.keyboard.press('ArrowUp')
  await expect(page.locator('nav.tabs [data-tab="snes"]')).toBeFocused()
})

test('controle PlayStation: L1/R1 trocam console, ✕ abre o jogo, ○ volta', async ({ page, app }) => {
  await open(page, app)
  await app.pad('r1', 'playstation')
  await app.pad('r1', 'playstation')
  await expect(page.locator('nav.tabs [aria-current="page"]')).toHaveText(/^SNES/)
  await expect(page.locator('.help .glyph').first()).toHaveText('✕') // ajuda com os símbolos do PS
  await app.pad('down', 'playstation')
  await app.pad('right', 'playstation')
  await expect(page.locator('#hero-title')).toHaveText('Super Mario World')
  await app.pad('south', 'playstation')
  await expect(page.locator('h1')).toHaveText('Super Mario World')
  await app.pad('east', 'playstation')
  await expect(page.locator('#hero-title')).toBeVisible()
  await app.pad('l1', 'playstation')
  await expect(page.locator('nav.tabs [aria-current="page"]')).toHaveText(/^NES/)
})

test('favoritar com F aparece na fileira de favoritos do Início', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('ArrowDown')
  const first = await focused(page)
  await page.keyboard.press('f')
  await expect(page.locator('.toasts')).toContainText('nos favoritos')
  await expect(page.locator('.shelf h2', { hasText: 'Favoritos' })).toBeVisible()
  await expect(page.locator(`.shelf [data-game="${first.game}"] .badge`).first()).toBeVisible()
  await page.reload()
  await expect(page.locator('.shelf h2', { hasText: 'Favoritos' })).toBeVisible()
})

test('jogar pelo destaque abre o emulador e volta com aviso', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('ArrowRight') // NES
  await page.keyboard.press('ArrowDown')
  await page.keyboard.press('ArrowUp') // botão Jogar
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  await expect(page.locator('.toasts')).toContainText('Super Mario Bros. 3')
  expect(app.calls()).toMatch(/retroarch\|--fullscreen -L .*mesen_libretro\.so .*Super Mario Bros\. 3\.zip/)
  // script de fim de jogo (é assim que o backup do solomon dispara)
  await expect.poll(() => (app.exists('hook.log') ? app.read('hook.log') : '')).toMatch(/Super Mario Bros\. 3\.zip\|Super Mario Bros\. 3\|nes\|\d+\|/)
  await expect(page.locator('.shelf h2', { hasText: 'Continuar jogando' })).toBeVisible({ timeout: 3000 }).catch(async () => {
    // a fileira de recentes só aparece no Início
    await page.keyboard.press('q')
    await expect(page.locator('.shelf h2', { hasText: 'Continuar jogando' })).toBeVisible()
  })
})
