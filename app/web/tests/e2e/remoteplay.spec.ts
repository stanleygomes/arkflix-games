// Jogar em outro aparelho: Sunshine falso (nada instala nem roda de verdade).
import { axe, expect, open, openItem, test } from './fixtures'

test('jogar em outro aparelho: instala, liga e mostra o IP pro Moonlight', async ({ page, app }) => {
  await open(page, app)
  await openItem(page, 'Jogar em outro aparelho')
  await expect(page.getByRole('heading', { name: 'Jogar em outro aparelho' })).toBeVisible()
  const state = page.getByTestId('remoteplay-state')
  await expect(state).toHaveText('Não instalado')
  // um botão principal por vez
  const install = page.getByRole('button', { name: 'Instalar o Sunshine' })
  await expect(page.getByRole('button', { name: 'Ligar' })).toHaveCount(0)
  await axe(page, 'jogar em outro aparelho')

  await install.click()
  await expect(state).toHaveText('Instalado, desligado', { timeout: 10000 })
  await expect(page.locator('.toasts')).toContainText('Sunshine instalado')
  await page.getByRole('button', { name: 'Ligar' }).click()
  await expect(state).toHaveText('Ligado')
  await expect(page.getByTestId('remoteplay-ip')).toHaveText('192.168.0.42')
  await expect(page.locator('.steps li')).toHaveCount(3)
  await expect(page.getByText('https://localhost:47990/pin')).toBeVisible()
  await axe(page, 'jogar em outro aparelho ligado')

  await page.getByRole('button', { name: 'Desligar' }).click()
  await expect(state).toHaveText('Instalado, desligado')
  await expect(page.getByRole('button', { name: 'Ligar' })).toBeVisible()
})
