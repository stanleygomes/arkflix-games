import { join } from 'node:path'
import { axe, expect, gameOption, open, test } from './fixtures'

test('editor de metadados: edita, valida, aparece em todo lugar e restaura', async ({ page, app }) => {
  await open(page, app)
  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').click()
  await gameOption(page, 'Editar informações')
  await expect(page.getByRole('heading', { name: 'Editar: Super Mario World' })).toBeVisible()
  const nameField = page.getByRole('button', { name: /^Nome Super Mario World automático/ })
  await expect(nameField).toBeFocused()
  await axe(page, 'editor de metadados')

  await nameField.click()
  const dialog = page.getByRole('dialog', { name: 'Nome' })
  await dialog.getByLabel('Nome').fill('SMW clássico')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: /^Nome SMW clássico editado/ })).toBeVisible()

  await page.getByRole('button', { name: /^Ano 1990/ }).click()
  const year = page.getByRole('dialog', { name: /Ano/ })
  await year.getByLabel(/Ano/).fill('19x')
  await page.keyboard.press('Enter')
  await expect(year.getByRole('alert')).toContainText('4 dígitos')
  await year.getByLabel(/Ano/).fill('1991')
  await page.keyboard.press('Enter')
  await expect(page.getByRole('button', { name: /^Ano 1991 editado/ })).toBeVisible()

  await page.getByRole('radio', { name: '★★★★★' }).click()
  await page.getByRole('switch', { name: /Completo/ }).click()
  await expect(page.getByRole('switch', { name: /Completo/ })).toHaveAttribute('aria-checked', 'true')

  const cover = join(app.root, 'home/Games/roms/genesis/covers/Sonic The Hedgehog 2 (World) (Rev A).png')
  await page.getByRole('button', { name: 'Escolher imagem' }).click()
  await page.getByRole('dialog', { name: 'Caminho da imagem da capa' }).getByLabel('Caminho da imagem da capa').fill(cover)
  await page.getByRole('button', { name: 'Usar esta capa' }).click()
  await expect(page.locator('main')).toContainText('Escolhida:')

  const saved = JSON.parse(app.read(`${app.data}/metadata.json`))
  expect(Object.values(saved)[0]).toMatchObject({ name: 'SMW clássico', year: 1991, rating: 5, completed: true })

  await page.getByRole('button', { name: 'Pronto' }).click()
  await expect(page.locator('h1')).toHaveText('SMW clássico')
  await expect(page.locator('.facts')).toContainText('★★★★★')
  await expect(page.locator('.facts')).toContainText('Completo')

  await page.keyboard.press('Escape')
  await page.keyboard.press('/')
  await page.keyboard.type('classico')
  await expect(page.locator('.results .name')).toHaveText(['SMW clássico'])

  // restaurar só o nome, depois tudo
  await page.keyboard.press('Escape')
  await page.locator('.shelf [data-game][aria-label^="SMW clássico"]').first().click()
  await gameOption(page, 'Editar informações')
  await page.getByRole('button', { name: 'Restaurar nome automático' }).click()
  await expect(page.getByRole('button', { name: /^Nome Super Mario World automático/ })).toBeVisible()
  await page.getByRole('button', { name: 'Restaurar tudo' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Restaurar tudo' }).click()
  await expect(page.getByRole('button', { name: /^Ano 1990 automático/ })).toBeVisible()
  await expect(page.getByRole('switch', { name: /Completo/ })).toHaveAttribute('aria-checked', 'false')
})
