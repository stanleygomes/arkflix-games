import { axe, expect, goTab, open, test } from './fixtures'

test('listas: criar pela página do jogo, ver a fileira, renomear e apagar', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').click()
  await page.getByRole('button', { name: /^Listas/ }).click()
  const picker = page.getByRole('dialog', { name: 'Listas' })
  await expect(picker).toContainText('Nenhuma lista ainda')
  await axe(page, 'seletor de listas')

  await picker.getByRole('button', { name: 'Nova lista' }).click()
  const dialog = page.getByRole('dialog', { name: 'Nova lista' })
  await expect(dialog.getByLabel('Nova lista')).toBeFocused()
  await axe(page, 'diálogo de texto')
  await page.keyboard.press('Enter') // vazio: erro, não fecha
  await expect(dialog.getByRole('alert')).toHaveText('dê um nome pra lista')
  await page.keyboard.type('Plataforma raiz')
  await page.keyboard.press('Enter')
  await expect(dialog).toBeHidden()
  await expect(picker.getByRole('checkbox', { name: /Plataforma raiz/ })).toHaveAttribute('aria-checked', 'true')
  await picker.getByRole('button', { name: 'Pronto' }).click()
  await expect(page.getByRole('button', { name: 'Listas (1)' })).toBeVisible()
  const saved = JSON.parse(app.read(`${app.data}/lists.json`))
  expect(saved[0]).toMatchObject({ name: 'Plataforma raiz', games: [expect.any(String)] })

  await page.keyboard.press('Escape')
  const row = page.locator('.shelf', { has: page.locator('h2', { hasText: 'Plataforma raiz' }) })
  await expect(row.locator('[data-game]')).toHaveCount(1)
  await row.getByRole('button', { name: 'Ver tudo' }).click()
  await expect(page.getByRole('heading', { name: 'Plataforma raiz' })).toBeVisible()

  await page.keyboard.press('m') // opções da lista
  await page.getByRole('button', { name: 'Renomear lista' }).click()
  const rename = page.getByRole('dialog', { name: 'Renomear lista' })
  await rename.getByLabel('Renomear lista').fill('Mario')
  await rename.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByRole('heading', { name: 'Mario' })).toBeVisible()

  await page.keyboard.press('m') // opções da lista
  await page.getByRole('button', { name: 'Apagar lista' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Apagar' }).click()
  await page.keyboard.press('Escape')
  await expect(page.locator('.shelf h2', { hasText: 'Mario' })).toHaveCount(0)
  expect(JSON.parse(app.read(`${app.data}/lists.json`))).toEqual([])
})

test('nova lista só com o controle (teclado na tela)', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Mega Man X"]').click()
  await page.getByRole('button', { name: /^Listas/ }).click()
  await page.getByRole('button', { name: 'Nova lista' }).click()
  for (const key of ['a', 'c', 'a', 'o']) {
    await page.locator(`.osk [data-key="${key}"]`).focus()
    await app.pad('south')
  }
  await page.getByRole('dialog', { name: 'Nova lista' }).getByRole('button', { name: 'Criar lista' }).focus()
  await app.pad('south')
  await expect(page.getByRole('dialog', { name: 'Listas' }).getByRole('checkbox', { name: /acao/ })).toHaveAttribute('aria-checked', 'true')
})

test('lista automática: salva os filtros, aparece no Início e não aceita jogo na mão', async ({ page, app }) => {
  await open(page, app)
  await goTab(page, 'snes')
  await page.getByRole('button', { name: 'Ver tudo' }).click()
  await page.keyboard.press('m')
  const dialog = page.getByRole('dialog', { name: 'Opções da lista' })
  await expect(dialog.getByRole('button', { name: 'Salvar como lista automática' })).toHaveCount(0) // sem filtro
  await dialog.getByRole('checkbox', { name: 'Ação' }).click()
  await dialog.getByRole('radio', { name: 'Ano (mais novos)' }).click()
  await dialog.getByRole('button', { name: 'Salvar como lista automática' }).click()
  const name = page.getByRole('dialog', { name: 'Nome da lista automática' })
  await expect(name.getByLabel('Nome da lista automática')).toHaveValue('Super Nintendo: Ação')
  await name.getByLabel('Nome da lista automática').fill('Ação no SNES')
  await page.keyboard.press('Enter')
  await expect(name).toBeHidden()
  const saved = JSON.parse(app.read(`${app.data}/lists.json`))
  expect(saved[0]).toMatchObject({ name: 'Ação no SNES', games: [], rule: { source: 'system:snes', filters: ['genre:Ação'], sort: 'yearDesc' } })

  await page.keyboard.press('Escape')
  await goTab(page, 'home')
  const row = page.locator('.shelf', { has: page.locator('h2', { hasText: 'Ação no SNES' }) })
  await expect(row.locator('h2')).toContainText('(lista automática)')
  await expect(row.locator('[data-game]')).toHaveCount(1)
  await row.getByRole('button', { name: 'Ver tudo' }).click()
  await expect(page.getByRole('heading', { name: 'Ação no SNES' })).toBeVisible()
  await expect(page.locator('.head')).toContainText('Lista automática')
  await expect(page.locator('.grid .name')).toHaveText(['Mega Man X'])
  await axe(page, 'lista automática')

  await page.keyboard.press('m')
  await expect(dialog.getByRole('button', { name: 'Salvar como lista automática' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Renomear lista' }).click()
  const rename = page.getByRole('dialog', { name: 'Renomear lista' })
  await rename.getByLabel('Renomear lista').fill('Ação')
  await rename.getByRole('button', { name: 'Salvar' }).click()
  await expect(page.getByRole('heading', { name: 'Ação', exact: true })).toBeVisible()

  // o seletor de listas da página do jogo só mostra as manuais
  await page.locator('.grid [data-game]').first().click()
  await page.getByRole('button', { name: /^Listas/ }).click()
  await expect(page.getByRole('dialog', { name: 'Listas' })).toContainText('Nenhuma lista ainda')
})
