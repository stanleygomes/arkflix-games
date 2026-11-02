import { axe, expect, test } from './fixtures'

// Simula o app desktop: um "Wails" falso que repassa cada método pro servidor
// de teste e registra os de janela (que no navegador não existem).
async function asDesktop(page: import('@playwright/test').Page) {
  await page.addInitScript(() => {
    const calls: string[] = []
    let fullscreen = true
    const local: Record<string, () => unknown> = {
      Minimize: () => calls.push('Minimize'),
      Quit: () => calls.push('Quit'),
      ToggleFullscreen: () => ((fullscreen = !fullscreen), calls.push('ToggleFullscreen')),
      IsFullscreen: () => fullscreen,
    }
    const remote = async (name: string, args: unknown[]) => {
      const res = await fetch(`/api/call/${name}`, { method: 'POST', headers: { 'Content-Type': 'application/json', 'X-Arkflix': '1' }, body: JSON.stringify(args) })
      const body = await res.json()
      if (!res.ok) throw body.error
      return name === 'Bootstrap' ? { ...body.result, mode: 'desktop' } : body.result
    }
    const service = new Proxy(
      {},
      {
        // "then" fora: senão o Proxy parece uma Promise
        get: (_, name: string) => (name === 'then' ? undefined : (...args: unknown[]) => (local[name] ? Promise.resolve(local[name]()) : remote(name, args))),
      },
    )
    const source = new EventSource('/api/events')
    const w = window as unknown as Record<string, unknown>
    w.__calls = calls
    w.go = { core: { Service: service } }
    w.runtime = {
      EventsOn: (name: string, cb: (d: unknown) => void) => {
        const h = (e: MessageEvent) => {
          const ev = JSON.parse(e.data)
          if (ev.name === name) cb(ev.data)
        }
        source.addEventListener('message', h)
        return () => source.removeEventListener('message', h)
      },
      EventsOff: () => {},
    }
  })
}

const calls = (page: import('@playwright/test').Page) => page.evaluate(() => (window as unknown as { __calls: string[] }).__calls)

test('menu de energia: minimizar, sair da tela cheia e fechar com confirmação', async ({ page, app }) => {
  await asDesktop(page)
  await page.goto(app.url)
  // energia fica fixa na barra do topo, sempre à mão
  const power = page.getByRole('button', { name: 'Energia', exact: true })
  await power.click()
  const menu = page.getByRole('dialog', { name: 'Energia' })
  await expect(menu.getByRole('button', { name: 'Minimizar' })).toBeFocused()
  await axe(page, 'menu de energia')
  await menu.getByRole('button', { name: 'Minimizar' }).click()
  await expect(menu).toHaveCount(0)

  await power.click()
  await menu.getByRole('button', { name: 'Sair da tela cheia' }).click()
  await power.click()
  await expect(menu.getByRole('button', { name: 'Tela cheia', exact: true })).toBeVisible()

  await menu.getByRole('button', { name: 'Fechar o app' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Cancelar' }).click()
  expect(await calls(page)).not.toContain('Quit')
  await menu.getByRole('button', { name: 'Fechar o app' }).click()
  await page.getByRole('alertdialog').getByRole('button', { name: 'Fechar' }).click()
  await expect.poll(() => calls(page)).toEqual(['Minimize', 'ToggleFullscreen', 'Quit'])
})

test('no navegador não há menu de energia', async ({ page, app }) => {
  await page.goto(app.url)
  await expect(page.locator('main')).toBeVisible()
  await expect(page.getByRole('button', { name: /^Energia/ })).toHaveCount(0)
})

test.describe('com animação', () => {
  test.use({ reducedMotion: 'no-preference' })
  test('ligar o app desktop mostra a tela de boot, e um botão pula', async ({ page, app }) => {
    await asDesktop(page)
    await page.goto(app.url)
    await expect(page.locator('.boot')).toBeVisible()
    await page.keyboard.press('Enter')
    await expect(page.locator('.boot')).toHaveCount(0)
    await page.goto(app.url)
    await expect(page.locator('.boot')).toBeVisible()
    await expect(page.locator('.boot')).toHaveCount(0, { timeout: 4000 }) // some sozinha
  })
})
