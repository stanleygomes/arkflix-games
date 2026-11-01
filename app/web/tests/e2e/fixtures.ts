// Fixture "app": servidor Go próprio por teste, numa pasta isolada.
import AxeBuilder from '@axe-core/playwright'
import { test as base, expect, type Page } from '@playwright/test'
import { spawn } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import { BIN, DIST } from './paths'

export interface App {
  url: string
  root: string
  /** Aperta e solta um botão do controle (como se viesse do evdev). */
  pad: (button: string, family?: string) => Promise<void>
  /** Chamadas que os emuladores falsos receberam. */
  calls: () => string
  /** Lê um arquivo do ambiente falso (caminho relativo à raiz). */
  read: (rel: string) => string
  exists: (rel: string) => boolean
  data: string
}

type Options = { fakeExit: string; fakeSleep: string }

export const test = base.extend<Options & { app: App }>({
  fakeExit: ['0', { option: true }],
  fakeSleep: ['0.8', { option: true }],
  app: async ({ page, fakeExit, fakeSleep }, use, info) => {
    const root = info.outputPath('fx')
    const proc = spawn(BIN, ['--web', '127.0.0.1:0', '--fixture', root, '--test', '--assets', DIST], {
      env: { ...process.env, ARKFLIX_FAKE_EXIT: fakeExit, ARKFLIX_FAKE_SLEEP: fakeSleep },
    })
    let log = ''
    const url = await new Promise<string>((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error(`servidor não subiu: ${log}`)), 10_000)
      proc.stdout.on('data', (d: Buffer) => {
        log += String(d)
        const m = /em (http:\/\/\S+)/.exec(log)
        if (m) {
          clearTimeout(timer)
          resolve(m[1])
        }
      })
      proc.stderr.on('data', (d: Buffer) => (log += String(d)))
      proc.on('exit', (code: number | null) => reject(new Error(`servidor saiu (${code}): ${log}`)))
    })
    const pad = async (button: string, family = 'xbox') => {
      for (const pressed of [true, false]) {
        const res = await page.request.post(`${url}/api/test/pad`, { data: { button, pressed, family, name: 'Controle de teste' } })
        expect(res.ok()).toBeTruthy()
      }
    }
    const app: App = {
      url,
      root,
      pad,
      data: 'home/.config/arkflix-games',
      calls: () => (existsSync(join(root, 'bin/calls.log')) ? readFileSync(join(root, 'bin/calls.log'), 'utf8') : ''),
      read: (rel) => readFileSync(join(root, rel), 'utf8'),
      exists: (rel) => existsSync(join(root, rel)),
    }
    await use(app)
    proc.kill()
  },
})

export { expect }

/** Abre o app e espera a primeira tela. */
export async function open(page: Page, app: App, path = '/') {
  await page.goto(app.url + path)
  await expect(page.locator('main')).toBeVisible()
  await expect(page.locator('[data-nav]:focus')).toHaveCount(1)
}

/** Varredura de acessibilidade (axe, WCAG 2.1 A/AA). Zero violações. */
export async function axe(page: Page, label: string) {
  const res = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
  const found = res.violations.map((v) => `${v.id} (${v.impact}): ${v.nodes.map((n) => n.target.join(' ')).slice(0, 3).join(' | ')}`)
  expect(found, `acessibilidade em ${label}`).toEqual([])
}

export async function focused(page: Page) {
  return page.evaluate(() => {
    const el = document.activeElement as HTMLElement | null
    return { text: el?.textContent?.trim() ?? '', label: el?.getAttribute('aria-label') ?? '', game: el?.dataset.game ?? '' }
  })
}
