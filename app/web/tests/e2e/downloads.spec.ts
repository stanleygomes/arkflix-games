import { axe, expect, open, openItem, settings, test, type App, pickItem } from './fixtures'
import type { Page } from '@playwright/test'

// RomM falso lento: o DKC (~130 bytes) leva uns 13 s, dá pra pausar no meio
test.use({ rommSlow: '1500' })

async function connect(page: Page, app: App) {
  await settings(page)
  await pickItem(page, 'Servidor RomM')
  await page.getByRole('button', { name: /^Endereço/ }).click()
  await page.getByRole('dialog', { name: 'Endereço do servidor' }).getByLabel('Endereço do servidor').fill(`${app.url}/fake-romm`)
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: /^Usuário/ }).click()
  await page.getByRole('dialog', { name: 'Usuário do RomM' }).getByLabel('Usuário do RomM').fill('stanley')
  await page.keyboard.press('Enter')
  await page.getByRole('button', { name: 'Entrar' }).click()
  await page.getByRole('dialog', { name: 'Senha do RomM' }).getByLabel('Senha do RomM').fill('senha123')
  await page.keyboard.press('Enter')
  await expect(page.getByText(/3 jogos no servidor/).first()).toBeVisible()
  await page.keyboard.press('Escape') // Você
  await page.keyboard.press('Escape') // Início
  await page.locator('[data-view="store"]').click() // os do servidor ficam na Loja
}

async function download(page: Page, name: string) {
  await page.locator(`.shelf [data-game][aria-label^="${name}"]`).first().click()
  await page.getByRole('button', { name: /Baixar do servidor/ }).click()
}

test('fila de downloads: um por vez, pausar, continuar e opções', async ({ page, app }) => {
  test.setTimeout(90_000)
  await open(page, app)
  await connect(page, app)

  await download(page, 'Donkey Kong Country')
  await page.keyboard.press('Escape')
  await download(page, 'Spyro the Dragon')
  await expect(page.locator('main .downloading')).toContainText('Na fila')
  await page.keyboard.press('Escape')

  await openItem(page, 'Central')
  const dkc = page.locator('li.task', { hasText: 'Baixando Donkey Kong Country' })
  const spyro = page.locator('li.task', { hasText: 'Baixando Spyro the Dragon' })
  await expect(spyro).toContainText('na fila')
  await expect(spyro.getByRole('button')).toHaveText(['Cancelar']) // na fila: só cancelar
  await expect(dkc).toContainText('em andamento')

  // pausar no meio: o .part fica e o próximo da fila começa
  await dkc.getByRole('button', { name: 'Pausar' }).click()
  await expect(dkc).toContainText('pausado')
  await expect(dkc.getByRole('button', { name: 'Continuar' })).toBeVisible()
  expect(app.exists('home/Games/roms/snes/Donkey Kong Country (U).zip.part')).toBe(true)
  await expect(spyro).toContainText(/em andamento|pronto/)
  await axe(page, 'central com download pausado')

  await dkc.getByRole('button', { name: 'Continuar' }).click()
  await expect(spyro).toContainText('pronto', { timeout: 20_000 })
  await expect(dkc).toContainText('pronto', { timeout: 30_000 })
  expect(app.exists('home/Games/roms/snes/Donkey Kong Country (U).zip')).toBe(true)
  expect(app.exists('home/Games/roms/snes/Donkey Kong Country (U).zip.part')).toBe(false)
  expect(app.exists('home/Games/roms/psx/Spyro the Dragon (USA)/Spyro the Dragon (USA).bin')).toBe(true)

  // opções: velocidade e madrugada, num diálogo só
  await page.getByRole('button', { name: 'Opções de download' }).click()
  const dialog = page.getByRole('dialog', { name: 'Opções de download' })
  await expect(dialog.getByRole('radio', { name: 'Sem limite' })).toHaveAttribute('aria-checked', 'true')
  await axe(page, 'opções de download')
  await dialog.getByRole('radio', { name: '5 MB/s' }).click()
  await expect(dialog.getByRole('radio', { name: '5 MB/s' })).toHaveAttribute('aria-checked', 'true')
  await dialog.getByRole('switch', { name: 'Só de madrugada' }).click()
  await expect(dialog.getByRole('switch', { name: 'Só de madrugada' })).toHaveAttribute('aria-checked', 'true')
  await dialog.getByRole('button', { name: 'Fechar' }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Opções de download' })).toBeFocused()
  const saved = JSON.parse(app.read(`${app.data}/settings.json`))
  expect(saved.downloads).toEqual({ limitKBps: 5120, nightOnly: true })

  await page.reload()
  await openItem(page, 'Central')
  await page.getByRole('button', { name: 'Opções de download' }).click()
  await expect(page.getByRole('dialog', { name: 'Opções de download' }).getByRole('radio', { name: '5 MB/s' })).toHaveAttribute('aria-checked', 'true')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog', { name: 'Opções de download' })).toHaveCount(0)
})
