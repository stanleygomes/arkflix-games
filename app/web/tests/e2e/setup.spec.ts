import { axe, expect, test } from './fixtures'

test.describe('primeira vez', () => {
  test.use({ firstRun: '1' })

  test('app novo abre na configuração do perfil: nome e personagem', async ({ page, app }) => {
    await page.goto(app.url)
    await expect(page.getByRole('heading', { name: 'Boas-vindas! Monte seu perfil' })).toBeVisible()
    const start = page.getByRole('button', { name: 'Começar' })
    await expect(start).toBeDisabled()
    await page.keyboard.press('Escape') // não pula
    await expect(page.getByRole('heading', { name: 'Boas-vindas! Monte seu perfil' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Voltar' })).toHaveCount(0)

    const avatars = page.getByRole('radiogroup', { name: 'Avatar' }).getByRole('radio')
    expect(await avatars.count()).toBeGreaterThan(40)
    await expect(page.getByRole('radio', { name: 'Mario' })).toHaveAttribute('aria-checked', 'true')
    await page.getByRole('radio', { name: 'Sonic' }).click()
    const img = page.locator('.preview img')
    await expect(img).toHaveAttribute('src', '/avatars/sonic.png')
    await expect.poll(() => img.evaluate((i: HTMLImageElement) => i.naturalWidth)).toBe(64)
    const broken = await page.locator('.choices img').evaluateAll((imgs) => imgs.filter((i) => (i as HTMLImageElement).naturalWidth === 0).map((i) => i.getAttribute('src')))
    expect(broken, 'todo personagem tem imagem').toEqual([])

    await page.getByRole('button', { name: /^Nome/ }).click()
    await page.locator('#prompt-input').fill('Stanley')
    await page.keyboard.press('Enter')
    await axe(page, 'primeira vez')
    await start.click()
    await expect(page.getByRole('button', { name: 'Trocar perfil (Stanley)' })).toBeVisible()
    expect(JSON.parse(app.read(`${app.data}/profiles.json`))[0]).toMatchObject({ id: 'main', name: 'Stanley', avatar: 'sonic' })

    await page.reload()
    await expect(page.getByRole('button', { name: 'Trocar perfil (Stanley)' })).toBeVisible()
    await expect(page.getByRole('heading', { name: /Boas-vindas/ })).toHaveCount(0)
  })
})
