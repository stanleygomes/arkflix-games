import { axe, expect, open, test } from './fixtures'

async function openGame(page: import('@playwright/test').Page, app: import('./fixtures').App, name: string) {
  await open(page, app)
  await page.locator(`.shelf [data-game][aria-label^="${name}"]`).first().click()
  await expect(page.locator('h1')).toHaveText(name)
}

test('página do jogo: metadados do ES-DE, saves com miniatura e acessibilidade', async ({ page, app }) => {
  await openGame(page, app, 'Super Mario World')
  await expect(page.locator('.facts')).toContainText('1990')
  await expect(page.locator('.facts')).toContainText('1-2 jogadores')
  await expect(page.locator('.summary')).toContainText('Yoshi')
  const states = page.locator('.states [data-state]')
  await expect(states).toHaveCount(2)
  await expect(states.first()).toContainText('Automático')
  const loaded = await page.locator('.states img').evaluateAll((imgs) =>
    Promise.all((imgs as HTMLImageElement[]).map((i) => (i.complete ? i.naturalWidth : new Promise((r) => (i.onload = () => r(i.naturalWidth)))))),
  )
  expect(loaded.every((w) => (w as number) > 0)).toBeTruthy()
  await expect(page.locator('.files')).toContainText('Super Mario World (U) [!].srm')
  await expect(page.getByRole('button', { name: 'Continuar de onde parei' })).toBeVisible()
  await axe(page, 'página do jogo')
})

test('carregar o slot 1 passa --entryslot=1 pro RetroArch', async ({ page, app }) => {
  await openGame(page, app, 'Super Mario World')
  await page.getByRole('button', { name: /Carregar Slot 1/ }).click()
  await expect(page.getByRole('dialog', { name: /Jogando Super Mario World/ })).toBeVisible()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  expect(app.calls()).toContain('--entryslot=1')
})

test('continuar de onde parei liga o carregamento automático', async ({ page, app }) => {
  await openGame(page, app, 'Super Mario World')
  await page.getByRole('button', { name: 'Continuar de onde parei' }).click()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  expect(app.calls()).toMatch(/retroarch-input\.cfg\|.*arkflix-resume\.cfg/)
})

test('apagar ponto salvo pede confirmação e some do disco', async ({ page, app }) => {
  await openGame(page, app, 'Super Mario World')
  const slot1 = page.getByRole('button', { name: /Carregar Slot 1/ })
  await slot1.focus()
  await page.keyboard.press('f')
  const dialog = page.getByRole('alertdialog')
  await expect(dialog).toContainText('Apagar o ponto salvo "Slot 1"')
  await axe(page, 'diálogo de confirmação')
  await page.keyboard.press('Escape') // cancela
  await expect(dialog).toBeHidden()
  await expect(page.locator('.states [data-state]')).toHaveCount(2)

  await slot1.focus()
  await page.keyboard.press('f')
  await dialog.getByRole('button', { name: 'Apagar' }).click()
  await expect(page.locator('.states [data-state]')).toHaveCount(1)
  const states = 'home/.var/app/org.libretro.RetroArch/config/retroarch/states'
  expect(app.exists(`${states}/Super Mario World (U) [!].state1`)).toBeFalsy()
  expect(app.exists(`${states}/Super Mario World (U) [!].state1.png`)).toBeFalsy()
  expect(app.exists(`${states}/Super Mario World (U) [!].state.auto`)).toBeTruthy()
})

test('PSP: save state achado pelo código da ISO e carregado no PPSSPP', async ({ page, app }) => {
  await openGame(page, app, 'Lara Croft Tomb Raider - Legend')
  await expect(page.locator('.states [data-state]')).toHaveCount(1)
  await expect(page.locator('.files')).toContainText('ULUS10084DATA00')
  await page.getByRole('button', { name: /Carregar Slot 1/ }).click()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  expect(app.calls()).toMatch(/ppsspp\|--fullscreen --state=.*ULUS10084_1\.00_0\.ppst .*\.iso/)
})

test('PS2: miniatura vem de dentro do .p2s e memory card aparece como compartilhado', async ({ page, app }) => {
  await openGame(page, app, 'FIFA Soccer 07')
  await expect(page.locator('.states img')).toHaveCount(1)
  await expect(page.locator('.files')).toContainText('compartilhado entre jogos')
  await page.getByRole('button', { name: 'Jogar do início' }).click()
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  expect(app.calls()).toMatch(/pcsx2\|-batch -fullscreen -- .*FIFA Soccer 07 \(USA\)\.iso/)
})

test.describe('emulador com erro', () => {
  test.use({ fakeExit: '3', fakeSleep: '0.2' })
  test('mostra o erro em vez de fingir que deu certo', async ({ page, app }) => {
    await openGame(page, app, 'Mega Man X')
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.locator('.toasts')).toContainText('fechou com erro 3')
  })
})

test('contador de partidas e histórico de sessões', async ({ page, app }) => {
  await openGame(page, app, 'Mega Man X')
  await expect(page.locator('.facts')).toContainText('0 partidas')
  for (let i = 0; i < 2; i++) {
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
  }
  await expect(page.locator('.facts')).toContainText('2 partidas')
  const lines = app.read(`${app.data}/sessions.jsonl`).trim().split('\n').map((l) => JSON.parse(l))
  expect(lines).toHaveLength(2)
  expect(lines[0]).toMatchObject({ system: 'snes', name: 'Mega Man X', mode: 'start', code: 0 })
})
