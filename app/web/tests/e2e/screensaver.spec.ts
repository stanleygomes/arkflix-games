import type { Page } from '@playwright/test'
import { axe, expect, open, test, type App, settings } from './fixtures'

async function patchSettings(page: Page, app: App, fn: (s: Record<string, any>) => Record<string, any>) {
  const call = (m: string, args: unknown[] = []) =>
    page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
  const s = (await (await call('GetSettings')).json()).result
  expect((await call('SaveSettings', [fn(s)])).ok()).toBeTruthy()
  await expect.poll(async () => (await (await call('Bootstrap')).json()).result?.games?.length ?? 0).toBeGreaterThan(0)
}

const saver = (page: Page) => page.getByRole('dialog', { name: 'Protetor de tela' })


// estes efeitos só existem com animação ligada
test.use({ reducedMotion: 'no-preference' })

test('cobrinha liga depois de 5 min parado, joga sozinha e qualquer tecla só acorda', async ({ page, app }) => {
  await page.clock.install()
  await open(page, app)
  await page.clock.fastForward('04:00')
  await expect(saver(page)).toHaveCount(0)
  await page.clock.fastForward('01:05')
  await expect(saver(page)).toBeVisible()
  await expect(saver(page)).toHaveAttribute('data-mode', 'snake')
  await page.clock.runFor(60_000) // ~700 passos
  await expect(saver(page).locator('.score')).not.toContainText('SCORE 0000')
  await axe(page, 'protetor de tela')

  await page.keyboard.press('f') // seria "favoritar": aqui só acorda
  await expect(saver(page)).toHaveCount(0)
  await expect(page.locator('.shelf h2', { hasText: 'Favoritos' })).toHaveCount(0)

  // o controle também acorda sem agir
  await page.clock.fastForward('05:05')
  await expect(saver(page)).toBeVisible()
  await app.pad('a')
  await expect(saver(page)).toHaveCount(0)
  await expect(page.locator('h1')).toHaveCount(1) // continua no Início
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toHaveCount(0)
})

test.describe('com jogo aberto', () => {
  test.use({ fakeSleep: '15' })
  test('não liga enquanto o jogo roda', async ({ page, app }) => {
    await page.clock.install()
    await open(page, app)
    await page.getByRole('button', { name: 'Jogar', exact: true }).click()
    await expect(page.getByRole('dialog', { name: /Jogando/ })).toBeVisible()
    await page.clock.fastForward('06:00')
    await page.clock.runFor(2000)
    await expect(saver(page)).toHaveCount(0)
  })
})

test('vitrine com reduzir animações: capa parada, INSERT COIN e clique só acorda', async ({ page, app }) => {
  await patchSettings(page, app, (s) => ({
    ...s,
    a11y: { ...s.a11y, reduceMotion: true },
    screensaver: { enabled: true, minutes: 1, mode: 'showcase' },
  }))
  await page.clock.install()
  await open(page, app)
  await page.clock.fastForward('01:05')
  await expect(saver(page)).toHaveAttribute('data-mode', 'showcase')
  await expect(saver(page)).toContainText('INSERT COIN')
  const img = saver(page).locator('img')
  const first = await img.getAttribute('src')
  expect(first).toMatch(/^\/media\/cover\//)
  await page.clock.runFor(12_000)
  expect(await img.getAttribute('src')).toBe(first) // parada: não troca
  await axe(page, 'vitrine')
  await page.mouse.click(10, 10)
  await expect(saver(page)).toHaveCount(0)
  await expect(page.locator('h1')).toHaveCount(1)
})

test('configuração do protetor: minutos, estilo, desligar e ver agora', async ({ page, app }) => {
  await open(page, app)
  await settings(page)
  await page.getByRole('button', { name: 'Acessibilidade', exact: true }).click()
  await page.getByRole('group', { name: 'Minutos parado' }).getByRole('button', { name: '10 min' }).click()
  await page.getByRole('group', { name: 'Estilo do protetor' }).getByRole('button', { name: 'Vitrine' }).click()
  const file = `${app.data}/settings.json`
  await expect
    .poll(() => (app.exists(file) ? JSON.parse(app.read(file)).screensaver : null))
    .toMatchObject({ enabled: true, minutes: 10, mode: 'showcase' })
  await page.getByRole('button', { name: 'Ver agora' }).click()
  await expect(saver(page)).toHaveAttribute('data-mode', 'showcase')
  await page.keyboard.press('Escape')
  await expect(saver(page)).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Ver agora' })).toBeVisible() // continua nas configurações

  await page.getByRole('switch', { name: 'Ligar quando parado' }).click()
  await expect(page.getByRole('button', { name: 'Ver agora' })).toHaveCount(0)
  await expect.poll(() => JSON.parse(app.read(file)).screensaver.enabled).toBe(false)
})

test('descanso: depois de 15 min de protetor a tela apaga, e um botão acorda', async ({ page, app }) => {
  await page.clock.install()
  await open(page, app)
  await page.clock.fastForward('05:05')
  await expect(saver(page)).toBeVisible()
  await page.clock.fastForward('15:05')
  const rest = page.getByRole('dialog', { name: 'Descanso' })
  await expect(rest).toBeVisible()
  await expect(saver(page)).toHaveCount(0)
  await expect(page.locator('canvas')).toHaveCount(0) // nada animando
  await page.keyboard.press('Enter')
  await expect(rest).toHaveCount(0)
  await expect(page.getByRole('dialog', { name: /Jogando/ })).toHaveCount(0) // só acordou
})
