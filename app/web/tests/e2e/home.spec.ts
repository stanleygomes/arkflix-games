import { axe, expect, focused, goTab, open, test } from './fixtures'

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
  await goTab(page, 'snes')
  await expect(page.locator('nav.tabs [aria-current="page"]')).toHaveText(/^SNES/)
  // paleta do SNES (neon); a cor desliza, então espera ela chegar
  await expect.poll(() => page.evaluate(() => getComputedStyle(document.documentElement).getPropertyValue('--accent').trim())).toBe('rgb(184, 150, 255)')

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
  for (let i = 0; i < 2; i++) await app.pad('r1', 'playstation') // NES, SNES
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
  await goTab(page, 'nes')
  await page.keyboard.press('ArrowDown')
  await page.keyboard.press('ArrowUp') // botão Jogar
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  await expect(page.locator('.toasts')).toContainText('Super Mario Bros. 3')
  expect(app.calls()).toMatch(/retroarch\|--fullscreen --appendconfig=\S*retroarch-input\.cfg -L .*mesen_libretro\.so .*Super Mario Bros\. 3\.zip/)
  // script de fim de jogo (é assim que o backup do solomon dispara)
  await expect.poll(() => (app.exists('hook.log') ? app.read('hook.log') : '')).toMatch(/Super Mario Bros\. 3\.zip\|Super Mario Bros\. 3\|nes\|\d+\|/)
  // jogou: aparece a fileira "Continuar jogando" no Início
  await goTab(page, 'home')
  await expect(page.locator('.shelf h2', { hasText: 'Continuar jogando' })).toBeVisible()
})

test('fileiras do Início: biblioteca com Ver tudo, favoritos quando tem', async ({ page, app }) => {
  await open(page, app)
  await expect(page.locator('nav.tabs [data-tab^="collection:"]')).toHaveCount(0)
  await expect(page.locator('.shelf h2', { hasText: 'Favoritos' })).toHaveCount(0)
  const lib = page.locator('.shelf', { has: page.locator('h2', { hasText: 'Sua biblioteca' }) })
  await lib.getByRole('button', { name: 'Ver tudo' }).click()
  await expect(page.getByRole('heading', { name: 'Todos os jogos' })).toBeVisible()
  await expect(page.locator('.grid .sys').first()).toHaveText(/NES|SNES|MD|ARC|N64|PS1|PS2|PSP/)
  await axe(page, 'coleção todos os jogos')
  await page.locator('.grid [data-game]').first().focus()
  await page.keyboard.press('f') // favorita: a fileira Favoritos passa a existir
  await page.keyboard.press('Escape')
  await expect(page.locator('.shelf h2', { hasText: 'Favoritos' })).toBeVisible()
})
