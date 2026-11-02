// BIOS: conferência na aba Emuladores e aviso na página do jogo.
import { rmSync } from 'node:fs'
import { join } from 'node:path'
import { axe, expect, goTab, open, openItem, test, type App } from './fixtures'

const PS1 = 'home/.var/app/org.libretro.RetroArch/config/retroarch/system/scph5501.bin'
const PS2 = 'home/.var/app/net.pcsx2.PCSX2/config/PCSX2/bios/SCPH-39001.bin'
const remove = (app: App, rel: string) => rmSync(join(app.root, rel))

test('emuladores: mostra a BIOS de cada console e confere de novo', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Emuladores')
  const ps1 = page.locator('[data-bios="psx"]')
  const ps2 = page.locator('[data-bios="ps2"]')
  await expect(ps1).toContainText('ok: scph5501.bin')
  await expect(ps1).toContainText('retroarch/system')
  await expect(ps2).toContainText('ok: SCPH-39001.bin')
  await expect(page.locator('[data-bios="nds"]')).toContainText('opcional')
  await expect(page.locator('.bios-list li')).toHaveCount(4)
  await axe(page, 'BIOS nos emuladores')

  remove(app, PS2)
  await page.getByRole('button', { name: 'Conferir de novo' }).click()
  await expect(ps2).toContainText('falta BIOS do PS2')
  await expect(ps2).toContainText('PCSX2/bios')
  await expect(ps2).not.toHaveClass(/ok/)
  // sem desktop não abre o gerenciador, mas diz qual é a pasta
  await ps2.getByRole('button', { name: /Abrir pasta/ }).click()
  await expect(page.locator('.toasts')).toContainText('PCSX2/bios')
})

test('página do jogo avisa da BIOS que falta e o Jogar explica', async ({ page, app }) => {
  remove(app, PS1)
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="FIFA Soccer 07"]').first().click()
  await expect(page.locator('h1')).toHaveText('FIFA Soccer 07')
  await expect(page.locator('.bios-warn')).toHaveCount(0)

  await open(page, app)
  await goTab(page, 'psx')
  await page.locator('.shelf [data-game][aria-label^="Crash Bandicoot"]').first().click()
  await expect(page.locator('h1')).toHaveText('Crash Bandicoot')
  const warn = page.locator('.bios-warn')
  await expect(warn).toContainText('Falta a BIOS do PlayStation (scph5501.bin, scph5500.bin ou scph5502.bin)')
  await expect(warn).toContainText('retroarch/system')
  await axe(page, 'aviso de BIOS na página do jogo')
  await page.getByRole('button', { name: 'Jogar do início' }).click()
  await expect(page.locator('.toasts')).toContainText('Falta a BIOS do PlayStation (scph5501.bin etc.): coloque em')
  expect(app.calls()).toBe('')
})
