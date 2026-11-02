import { axe, expect, open, test, settings } from './fixtures'

async function openControls(page: import('@playwright/test').Page, app: import('./fixtures').App) {
  await open(page, app)
  await settings(page)
  await page.getByRole('button', { name: 'Controles', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'Controles' })).toBeVisible()
}

test('prévia acende o botão apertado e troca pro desenho do controle em uso', async ({ page, app }) => {
  await openControls(page, app)
  await axe(page, 'configurações: controles')
  await page.request.post(`${app.url}/api/test/pad`, { data: { button: 'south', pressed: true, family: 'playstation' } })
  await expect(page.locator('svg [data-button="south"]')).toHaveClass(/pressed/)
  await expect(page.locator('figcaption')).toHaveText('PlayStation')
  await expect(page.locator('.live')).toContainText('✕')
  await page.request.post(`${app.url}/api/test/pad`, { data: { button: 'south', pressed: false, family: 'playstation' } })
  await expect(page.locator('svg [data-button="south"]')).not.toHaveClass(/pressed/)
})

test('remapear no controle grava a config do RetroArch e avisa conflito', async ({ page, app }) => {
  await openControls(page, app)
  const cfg = `${app.data}/retroarch-input.cfg`
  expect(app.read(cfg)).toContain('input_player1_a_btn = "1"')

  await page.locator('[data-retropad="a"]').click()
  await expect(page.locator('.live')).toContainText('Aperte o botão pra A (direita)')
  // durante a captura, o botão não navega: vira o novo mapeamento
  await app.pad('south')
  await expect(page.locator('.toasts')).toContainText('A (direita) agora é A')
  await expect(page.getByRole('alert')).toContainText('south: a, b')
  await expect.poll(() => app.read(cfg)).toContain('input_player1_a_btn = "0"')

  await page.getByRole('button', { name: 'Restaurar padrão' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Restaurar' }).click()
  await expect.poll(() => app.read(cfg)).toContain('input_player1_a_btn = "1"')
  await expect(page.getByRole('alert')).toBeHidden()
})

test('teclado: remapear Start pra P', async ({ page, app }) => {
  await openControls(page, app)
  await page.getByRole('button', { name: 'Teclado' }).click()
  await page.locator('[data-retropad="start"]').click()
  await page.keyboard.press('KeyP')
  await expect(page.locator('[data-retropad="start"]')).toContainText('P')
  await expect.poll(() => app.read(`${app.data}/retroarch-input.cfg`)).toContain('input_player1_start = "p"')
  // Esc cancela sem mudar nada
  await page.locator('[data-retropad="select"]').click()
  await page.keyboard.press('Escape')
  await expect(page.locator('.toasts')).toContainText('Cancelado')
  expect(app.read(`${app.data}/retroarch-input.cfg`)).toContain('input_player1_select = "rshift"')
})

test('testar L1/R1 na prévia não troca de seção', async ({ page, app }) => {
  await openControls(page, app)
  await page.locator('[data-retropad="up"]').focus()
  await app.pad('r1')
  await app.pad('l1')
  await expect(page.getByRole('heading', { name: 'Controles' })).toBeVisible()
  await expect(page.locator('[data-retropad="up"]')).toBeFocused()
  // no teclado, E continua indo pra próxima seção
  await page.keyboard.press('e')
  await expect(page.getByRole('heading', { name: 'Acessibilidade' })).toBeVisible()
})
