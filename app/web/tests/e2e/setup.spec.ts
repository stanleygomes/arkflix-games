import { axe, expect, test } from './fixtures'

test.describe('primeira vez', () => {
  test.use({ firstRun: '1' })

  test('tela de boas-vindas: entra no RomM e o nome do perfil vem de lá', async ({ page, app }) => {
    await page.goto(app.url)
    await expect(page.getByRole('heading', { name: 'Bem-vindo!' })).toBeVisible()
    await page.keyboard.press('Escape') // sem "voltar" aqui
    await expect(page.getByRole('heading', { name: 'Bem-vindo!' })).toBeVisible()
    await axe(page, 'boas-vindas')

    const form = page.getByRole('form', { name: 'Entrar no servidor RomM' })
    await form.getByRole('button', { name: 'Entrar com o servidor' }).click()
    await expect(page.getByRole('alert')).toContainText('Preencha')

    const fill = async (field: RegExp, value: string) => {
      await form.getByRole('button', { name: field }).click()
      await page.locator('#prompt-input').fill(value)
      await page.keyboard.press('Enter')
    }
    await fill(/^Servidor/, `${app.url}/fake-romm`)
    await fill(/^Usuário/, 'stanley')
    await fill(/^Senha/, 'errada')
    await expect(form.getByRole('button', { name: /^Senha/ })).toContainText('••••••')
    await form.getByRole('button', { name: 'Entrar com o servidor' }).click()
    await expect(page.getByRole('alert')).toContainText('senha')
    await fill(/^Senha/, 'senha123')
    await form.getByRole('button', { name: 'Entrar com o servidor' }).click()

    await expect(page.getByRole('heading', { name: 'Boas-vindas! Monte seu perfil' })).toBeVisible()
    await expect(page.getByRole('button', { name: /^Nome/ })).toContainText('Stanley')
    await page.getByRole('button', { name: 'Começar' }).click()
    await expect(page.getByRole('button', { name: /^Você: Stanley/ })).toBeVisible()
    await expect(page.locator('[data-view="store"]')).toContainText(/\d/, { timeout: 10_000 }) // a Loja conta os do servidor
    await page.locator('[data-view="store"]').click()
    await expect(page.locator('.shelf [data-game][aria-label*="no servidor"]').first()).toBeVisible()
    expect(app.read(`${app.data}/romm.json`)).not.toContain('senha123')
  })

  test('app novo abre na configuração do perfil: nome e personagem', async ({ page, app }) => {
    await page.goto(app.url)
    await page.getByRole('button', { name: 'Jogar offline' }).click()
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
    await expect(page.getByRole('button', { name: /^Você: Stanley/ })).toBeVisible()
    expect(JSON.parse(app.read(`${app.data}/profiles.json`))[0]).toMatchObject({ id: 'main', name: 'Stanley', avatar: 'sonic' })

    await page.reload()
    await expect(page.getByRole('button', { name: /^Você: Stanley/ })).toBeVisible()
    await expect(page.getByRole('heading', { name: /Boas-vindas/ })).toHaveCount(0)
  })
})
