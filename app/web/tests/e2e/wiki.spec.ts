import { axe, expect, open, settings, test, gameOption } from './fixtures'

test('editor: buscar na internet preenche só o vazio e mostra a fonte', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Sonic The Hedgehog 2"]').first().click()
  await gameOption(page, 'Editar informações')
  await expect(page.getByRole('heading', { name: /^Editar: Sonic/ })).toBeVisible()
  const fetch = page.getByRole('button', { name: 'Buscar na internet' })
  await expect(fetch).toBeVisible()
  await axe(page, 'editor com busca na internet')

  await fetch.click()
  await expect(page.locator('.toasts')).toContainText('Preenchido: descrição, ano, gêneros')
  await expect(page.getByRole('button', { name: /^Ano 1992 editado/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /^Gêneros Plataforma editado/ })).toBeVisible()
  await expect(page.getByRole('button', { name: /^Descrição Sonic the Hedgehog 2 é um jogo/ })).toBeVisible()
  await expect(page.getByRole('status').filter({ hasText: 'Fonte:' })).toHaveText('Fonte: pt.wikipedia.org/wiki/Sonic_the_Hedgehog_2')
  await expect(fetch).toHaveCount(0) // nada mais vazio
  const saved = JSON.parse(app.read(`${app.data}/metadata.json`))
  expect(Object.values(saved)).toContainEqual(expect.objectContaining({ year: 1992, genres: ['Plataforma'] }))
})

test('configurações: completa em lote os jogos sem informação', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  const button = page.getByRole('button', { name: /^Completar informações que faltam \(\d+ jogos\)/ })
  await expect(button).toBeEnabled()
  await axe(page, 'configurações da biblioteca com completar informações')
  await button.click()
  await expect(page.locator('.toasts')).toContainText('Completando informações: 2 jogo(s) completado(s)', { timeout: 15000 })
  await expect(page.getByRole('button', { name: 'Completar informações que faltam (0 jogos)' })).toBeDisabled()

  await page.keyboard.press('Escape')
  await page.keyboard.press('Escape')
  await page.locator('.shelf [data-game][aria-label^="Crash Bandicoot"]').first().click()
  await expect(page.locator('main')).toContainText('Naughty Dog')
})
