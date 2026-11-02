import { expect, open, test } from './fixtures'

// Windows/macOS: o controle chega pela Gamepad API do WebView (aqui, simulada).
test.describe('controle pela interface', () => {
  test.use({ browserPads: '1' })

  test('navega e abre jogo com um controle da Gamepad API', async ({ page, app }) => {
    await page.addInitScript(() => {
      const buttons = Array.from({ length: 17 }, () => ({ pressed: false, value: 0 }))
      const gp = { index: 0, id: 'Xbox Wireless Controller (STANDARD GAMEPAD Vendor: 045e)', connected: true, mapping: 'standard', buttons, axes: [0, 0, 0, 0] }
      const w = window as unknown as { __press: (i: number, on: boolean) => void }
      w.__press = (i, on) => {
        buttons[i] = { pressed: on, value: on ? 1 : 0 }
      }
      Object.defineProperty(navigator, 'getGamepads', { value: () => [gp, null, null, null] })
    })
    await open(page, app)
    await expect(page.getByRole('img', { name: /1 controle\(s\): Xbox Wireless Controller/ })).toBeVisible()
    const press = async (i: number) => {
      await page.evaluate((b) => (window as unknown as { __press: (i: number, on: boolean) => void }).__press(b, true), i)
      await page.waitForTimeout(60)
      await page.evaluate((b) => (window as unknown as { __press: (i: number, on: boolean) => void }).__press(b, false), i)
      await page.waitForTimeout(60)
    }
    await press(13) // baixo: desce pra fileira de jogos
    await press(13)
    const focused = await page.evaluate(() => (document.activeElement as HTMLElement | null)?.dataset.game ?? '')
    expect(focused).not.toBe('')
    await press(0) // A: abre
    await expect(page.getByRole('button', { name: /Jogar do início|Baixar/ })).toBeVisible()
    await press(1) // B: volta
    await expect(page.locator('.shelf')).toBeVisible()
  })
})
