import { join } from 'node:path'
import { axe, expect, open, test } from './fixtures'

test('importar: detecta console, recusa o que não dá e copia pra biblioteca', async ({ page, app }) => {
  await open(page, app)
  await page.getByRole('button', { name: 'Importar jogos' }).click()
  await expect(page.locator('h1')).toHaveText('Importar jogos')
  const input = page.getByLabel('Caminho de um arquivo ou pasta')
  await expect(input).toBeFocused()

  for (const name of ['Tetris DX.gbc', 'Metal Slug X (USA).zip', 'Crazy Taxi.gdi']) {
    await input.fill(join(app.root, 'inbox', name))
    await input.press('Enter')
  }
  const items = page.locator('.items .item')
  await expect(items).toHaveCount(3)
  await expect(items.nth(0).locator('.console')).toHaveText(/GBC/)
  await expect(items.nth(1).locator('.console')).toHaveText(/PS1/)
  await expect(items.nth(1)).toContainText('vira pasta')
  await expect(items.nth(2)).toContainText('Dreamcast não é suportado')
  await axe(page, 'importação')

  await page.getByRole('button', { name: 'Importar 2 jogos' }).click()
  await expect(page.locator('.results li.ok')).toHaveCount(2)
  await expect(page.locator('.toasts')).toContainText('2 jogo(s) importado(s)')
  expect(app.exists('home/Games/roms/gbc/Tetris DX.gbc')).toBeTruthy()
  const cue = app.read('home/Games/roms/psx/Metal Slug X/Metal Slug X (USA).cue')
  expect(cue).toContain('FILE "Metal Slug X (USA).bin" BINARY') // caminho do Windows corrigido

  await page.keyboard.press('Escape')
  await expect(page.locator('nav.tabs [data-tab="gbc"]')).toBeVisible()
})

test('extensão desconhecida: escolhe o console na mão', async ({ page, app }) => {
  await open(page, app, '/')
  await page.getByRole('button', { name: 'Importar jogos' }).click()
  const input = page.getByLabel('Caminho de um arquivo ou pasta')
  await input.fill(join(app.root, 'inbox', 'nao-existe.nes'))
  await input.press('Enter')
  await expect(page.locator('.items .item')).toContainText('arquivo não encontrado')
  await expect(page.getByRole('button', { name: /Importar 0 jogos/ })).toBeDisabled()
})
