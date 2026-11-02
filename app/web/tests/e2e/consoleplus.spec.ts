// Recursos de console: emuladores pelo app, Bluetooth, conquistas e atualização.
import { axe, expect, open, openItem, test } from './fixtures'

test('emuladores: mostra o que tem e instala/atualiza pelo app', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Emuladores')
  await expect(page.getByRole('heading', { name: 'Emuladores' })).toBeVisible()
  await expect(page.locator('.emus li')).toHaveCount(4) // RetroArch, PPSSPP, PCSX2 e cores
  await axe(page, 'emuladores')
  await page.getByRole('button', { name: /Instalar o que falta|Atualizar tudo/ }).click()
  await expect(page.locator('.toasts')).toContainText(/pront/, { timeout: 15000 })
  await expect(page.locator('.emus li.ok')).toHaveCount(4)
})

test('bluetooth: procura e pareia o controle', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Bluetooth')
  await expect(page.getByRole('heading', { name: 'Bluetooth' })).toBeVisible()
  await expect(page.locator('.devs li', { hasText: '8BitDo Pro 2' })).toContainText('conectado')
  await page.getByRole('button', { name: 'Procurar controles' }).click()
  const xbox = page.locator('.devs li', { hasText: 'Xbox Wireless Controller' })
  await expect(xbox).toContainText('novo', { timeout: 10000 })
  await axe(page, 'bluetooth')
  await xbox.getByRole('button', { name: 'Parear' }).click()
  await expect(xbox).toContainText('conectado', { timeout: 10000 })
})

test('conquistas: entra no RetroAchievements e vê os troféus do jogo', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Conquistas')
  await page.getByRole('button', { name: 'Entrar no RetroAchievements' }).click()
  await page.getByRole('dialog').getByRole('textbox').fill('stanley')
  await page.getByRole('dialog').getByRole('button', { name: 'Continuar' }).click()
  await page.getByRole('dialog').getByLabel(/Senha/).fill('senha123')
  await page.getByRole('dialog').getByRole('button', { name: 'Entrar' }).click()
  await expect(page.getByText('Entrou como')).toBeVisible()
  expect(app.read(`${app.data}/cheevos.json`)).not.toContain('senha123')
  await axe(page, 'conquistas')

  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')
  // cartão no destaque do Início
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().focus()
  await expect(page.locator('.cards')).toContainText('1/3 conquistas')
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
  const box = page.locator('.cheevos')
  await expect(box).toContainText('1/3')
  await expect(box.locator('li.got')).toHaveCount(1)
  await axe(page, 'jogo com conquistas')
})

test.describe('com versão nova', () => {
  test.use({ releases: 'fake' })
  test('atualização: avisa atual → nova, atualiza num clique e reabre', async ({ page, app }) => {
    await open(page, app)
    await openItem(page, 'Atualizações')
    await expect(page.getByRole('heading', { name: 'Atualizações' })).toBeVisible()
    await page.getByRole('button', { name: 'Procurar atualizações' }).click()
    await expect(page.locator('.versions')).toHaveText(/v1\.0\.0\s*→\s*v99\.0\.0/)
    // aviso no topo, em qualquer tela com a barra
    await expect(page.getByRole('button', { name: 'Atualização disponível: v1.0.0 → v99.0.0' })).toBeVisible()
    await expect(page.locator('.toasts')).toContainText('Atualização disponível: v1.0.0 → v99.0.0')
    await axe(page, 'atualizações')
    await page.getByRole('button', { name: /e (reiniciar|atualizar)$/ }).click()
    await expect(page.getByRole('alertdialog', { name: 'Atualizando pra v99.0.0' })).toBeVisible({ timeout: 15000 })
    // no navegador "reabrir" é recarregar: volta pra interface
    await expect(page.getByRole('alertdialog', { name: /Atualizando/ })).toHaveCount(0, { timeout: 10000 })
    await expect(page.locator('#hero-title')).toBeVisible()
  })
})
