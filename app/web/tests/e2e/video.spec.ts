import type { Page } from '@playwright/test'
import { axe, expect, open, test, type App } from './fixtures'

async function setA11y(page: Page, app: App, patch: Record<string, boolean>) {
  const call = (m: string, args: unknown[] = []) =>
    page.request.post(`${app.url}/api/call/${m}`, { data: args, headers: { 'X-Arkflix': '1' } })
  const s = (await (await call('GetSettings')).json()).result
  expect((await call('SaveSettings', [{ ...s, a11y: { ...s.a11y, ...patch } }])).ok()).toBeTruthy()
  // salvar varre a biblioteca de novo: espera os jogos voltarem
  await expect.poll(async () => (await (await call('Bootstrap')).json()).result?.games?.length ?? 0).toBeGreaterThan(0)
}

const sonic = (page: Page) => page.locator('.shelf [data-game][aria-label^="Sonic"]').first()


// estes efeitos só existem com animação ligada
test.use({ reducedMotion: 'no-preference' })

test('vídeo do jogo toca mudo no destaque depois de 2 s e some ao trocar de jogo', async ({ page, app }) => {
  await open(page, app)
  await sonic(page).focus()
  await expect(page.locator('#hero-title')).toHaveText(/Sonic/)
  const video = page.locator('video.hero-video')
  await page.waitForTimeout(800)
  await expect(video).toHaveCount(0) // ainda não: espera 2 s parado
  await expect(video).toHaveCount(1, { timeout: 4000 })
  expect(await video.evaluate((v: HTMLVideoElement) => v.muted)).toBe(true)
  await expect.poll(() => video.evaluate((v: HTMLVideoElement) => v.currentTime), { timeout: 5000 }).toBeGreaterThan(0)
  await axe(page, 'início com vídeo')

  await page.locator('.shelf [data-game][aria-label^="Super Mario World"]').first().focus()
  await expect(video).toHaveCount(0)
})

test('reduzir animações e o interruptor de vídeo desligam; som é opcional', async ({ page, app }) => {
  await setA11y(page, app, { reduceMotion: true })
  await open(page, app)
  await sonic(page).focus()
  await page.waitForTimeout(2600)
  await expect(page.locator('video')).toHaveCount(0)

  await setA11y(page, app, { reduceMotion: false, videos: false })
  await open(page, app)
  await sonic(page).focus()
  await page.waitForTimeout(2600)
  await expect(page.locator('video')).toHaveCount(0)

  await setA11y(page, app, { videos: true, videoSound: true })
  await open(page, app)
  await sonic(page).focus()
  const video = page.locator('video.hero-video')
  await expect(video).toHaveCount(1, { timeout: 4000 })
  expect(await video.evaluate((v: HTMLVideoElement) => v.muted)).toBe(false)
})

test('interruptores de vídeo nas configurações', async ({ page, app }) => {
  await open(page, app)
  await page.keyboard.press('m')
  await page.getByRole('button', { name: 'Acessibilidade', exact: true }).click()
  const sw = page.getByRole('switch', { name: 'Vídeo dos jogos' })
  await expect(sw).toHaveAttribute('aria-checked', 'true')
  await expect(page.getByRole('switch', { name: 'Som dos vídeos' })).toHaveAttribute('aria-checked', 'false')
  await sw.click()
  const file = `${app.data}/settings.json`
  await expect.poll(() => (app.exists(file) ? JSON.parse(app.read(file)).a11y.videos : true), { timeout: 3000 }).toBe(false)
})
