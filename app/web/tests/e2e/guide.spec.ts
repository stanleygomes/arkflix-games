import { axe, expect, open, test } from './fixtures'

test('guia rápido: abre com G, atalhos e central com avisos', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('f') // favorita: gera um aviso
  await expect(page.getByRole('button', { name: /^Central: .*aviso novo/ })).toBeVisible()
  await page.keyboard.press('g')
  const guide = page.getByRole('dialog', { name: 'Jogador 1' })
  await expect(guide).toBeVisible()
  await expect(guide.getByRole('button', { name: /Início/ })).toBeFocused()
  await expect(guide.getByRole('button', { name: /Central/ })).toContainText('1 novo')
  await axe(page, 'guia rápido')
  await page.keyboard.press('Escape')
  await expect(guide).toBeHidden()

  await page.keyboard.press('g')
  await guide.getByRole('button', { name: /Central/ }).click()
  await expect(page.getByRole('heading', { name: 'Central' })).toBeVisible()
  await expect(page.locator('.notice')).toContainText('favoritos')
  await axe(page, 'central')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('button', { name: /^Central: .*aviso novo/ })).toHaveCount(0) // visto
})

test.describe('com jogo aberto', () => {
  test.use({ fakeSleep: '30' })
  test('guia por cima do jogo: fecha o jogo', async ({ page, app }) => {
    test.setTimeout(60_000)
    await open(page, app)
    await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
    await page.keyboard.press('g')
    const guide = page.getByRole('dialog', { name: 'Jogador 1' })
    await expect(guide.getByRole('button', { name: 'Voltar ao jogo' })).toBeFocused()
    await expect(guide.getByRole('button', { name: 'Salvar estado' })).toBeVisible()
    await expect(guide.getByText(/Jogando há \d+ min/)).toBeVisible()
    await axe(page, 'guia com jogo')
    // anotação do jogo, guardada por perfil
    await guide.getByRole('button', { name: 'Anotação' }).click()
    await page.getByRole('dialog', { name: 'Anotação deste jogo' }).getByRole('textbox').fill('a chave tá no castelo 3')
    await page.getByRole('dialog', { name: 'Anotação deste jogo' }).getByRole('button', { name: 'Salvar' }).click()
    await expect(guide.locator('.note')).toHaveText('a chave tá no castelo 3')
    await guide.getByRole('button', { name: 'Capturar tela' }).click()
    await expect(page.locator('.toasts')).toContainText('Captura salva')
    await guide.getByRole('button', { name: 'Fechar jogo' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Fechar jogo' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
    await expect(page.locator('.toasts')).toContainText('Super Mario World')
  })
})

test.describe('retomada rápida', () => {
  test.use({ fakeSleep: '30' })
  test('troca pra outro jogo recente pelo guia; capturas aparecem na página do jogo', async ({ page, app }) => {
    test.setTimeout(90_000) // abre e fecha três jogos
    await open(page, app)
    // Mega Man X jogado antes, pra entrar nos recentes
    await page.locator('.shelf [data-game][aria-label^="Mega Man X"]').first().click()
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
    await page.keyboard.press('g')
    await page.getByRole('button', { name: 'Fechar jogo' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Fechar jogo' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })
    await page.keyboard.press('Escape')
    await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
    await page.getByRole('button', { name: 'Jogar do início' }).click()
    await expect(page.getByRole('dialog', { name: 'Jogando Super Mario World' })).toBeVisible()
    await page.keyboard.press('g')
    const guide = page.getByRole('dialog', { name: 'Jogador 1' })
    await guide.getByRole('button', { name: /Mega Man X/ }).click()
    await expect(page.getByRole('dialog', { name: 'Jogando Mega Man X' })).toBeVisible({ timeout: 10000 })
    await page.keyboard.press('g')
    await guide.getByRole('button', { name: 'Fechar jogo' }).click()
    await page.getByRole('alertdialog').getByRole('button', { name: 'Fechar jogo' }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeHidden({ timeout: 8000 })

    // captura e gravação do RetroArch (simuladas: arquivos na pasta do perfil)
    app.write(`${app.data}/media/screenshots/Super Mario World (U) [!]-251102-101010.png`, 'png')
    app.write(`${app.data}/media/videos/Super Mario World (U) [!]-2025_11_02.mkv`, 'mkv')
    await page.keyboard.press('Escape') // da página do jogo pro Início
    await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().click()
    await expect(page.locator('.strip')).toContainText('Captura de')
    await expect(page.locator('.strip')).toContainText('Gravação de')
    await expect(page.locator('.note-box')).toHaveCount(0)
  })
})
