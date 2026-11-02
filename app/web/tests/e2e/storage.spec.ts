import { axe, expect, open, test, settings, pickItem } from './fixtures'

test('armazenamento: disco, jogos por console e saves', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await pickItem(page, 'Armazenamento')
  await expect(page.getByRole('heading', { name: /livres de/ })).toBeVisible()
  const bar = page.getByRole('img', { name: /livres de .*Jogos: .*Saves: / })
  await expect(bar).toBeVisible()
  for (const label of ['Jogos', 'Saves', 'App e capas', 'Outros programas e arquivos', 'Livre']) {
    await expect(page.locator('.legend').getByText(label, { exact: true })).toBeVisible()
  }
  await expect(page.getByRole('heading', { name: /Jogos por console · 9 jogos/ })).toBeVisible()
  await expect(page.locator('.rows').first()).toContainText('Super Nintendo')
  await expect(page.getByRole('heading', { name: /^Saves · / })).toBeVisible()
  await expect(page.getByText('RetroArch (save states)')).toBeVisible()
  await axe(page, 'armazenamento')
})

test('armazenamento: mover os jogos de um console pra outra pasta', async ({ page, app }) => {
  const call = async (m: string, ...args: unknown[]) => {
    const res = await page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
    expect(res.ok()).toBeTruthy()
    return (await res.json()).result
  }
  const cfg = await call('GetSettings')
  for (const d of cfg.libraryDirs) d.writable = true // "Jogos antigos" vira gravável
  await call('SaveSettings', cfg)

  await open(page, app)
  await settings(page)
  await pickItem(page, 'Armazenamento')
  await page.getByRole('button', { name: 'Mover jogos de Super Nintendo' }).click()
  const dialog = page.getByRole('dialog', { name: 'Mover jogos de Super Nintendo' })
  await expect(dialog).toBeVisible()
  await expect(dialog).toContainText('2 jogos')
  await expect(dialog.getByRole('radio', { name: /Jogos antigos/ })).toHaveAttribute('aria-checked', 'true')
  await axe(page, 'mover jogos')
  await dialog.getByRole('button', { name: 'Mover', exact: true }).click()
  await expect(dialog).toBeHidden()
  await expect(page.getByText(/Movendo jogos de Super Nintendo: 2 jogo\(s\) movido\(s\)/)).toBeVisible()
  expect(app.exists('home/Jogos antigos/snes/Super Mario World (U) [!].zip')).toBeTruthy()
  expect(app.exists('home/Games/roms/snes/Super Mario World (U) [!].zip')).toBeFalsy()
  // a lista é medida de novo: o console continua lá, agora na outra pasta
  await expect(page.locator('.rows').first()).toContainText('Super Nintendo')
})

test('armazenamento: com uma pasta só, nada de mover', async ({ page, app }) => {
  const call = async (m: string, ...args: unknown[]) => {
    const res = await page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
    return (await res.json()).result
  }
  const cfg = await call('GetSettings')
  cfg.libraryDirs = cfg.libraryDirs.slice(0, 1)
  await call('SaveSettings', cfg)
  await open(page, app)
  await settings(page)
  await pickItem(page, 'Armazenamento')
  await expect(page.locator('.rows').first()).toContainText('Super Nintendo')
  await expect(page.getByRole('button', { name: /^Mover jogos/ })).toHaveCount(0)
})
