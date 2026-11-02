// Fixture "app": servidor Go próprio por teste, numa pasta isolada.
import AxeBuilder from '@axe-core/playwright'
import { test as base, expect, type Page } from '@playwright/test'
import { spawn } from 'node:child_process'
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, join } from 'node:path'
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
  write: (rel: string, body: string) => void
  data: string
}

type Options = { fakeExit: string; fakeSleep: string; firstRun: string; browserPads: string; releases: string }

export const test = base.extend<Options & { app: App }>({
  fakeExit: ['0', { option: true }],
  fakeSleep: ['0.8', { option: true }],
  firstRun: ['0', { option: true }],
  browserPads: ['0', { option: true }],
  releases: ['', { option: true }],
  app: async ({ page, fakeExit, fakeSleep, firstRun, browserPads, releases }, use, info) => {
    const root = info.outputPath('fx')
    const proc = spawn(BIN, ['--web', '127.0.0.1:0', '--fixture', root, '--test', '--assets', DIST], {
      env: { ...process.env, ARKFLIX_FAKE_EXIT: fakeExit, ARKFLIX_FAKE_SLEEP: fakeSleep, ARKFLIX_FIXTURE_FIRST_RUN: firstRun, ARKFLIX_FIXTURE_BROWSER_PADS: browserPads, ARKFLIX_FIXTURE_RELEASES: releases },
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
      write: (rel, body) => {
        mkdirSync(dirname(join(root, rel)), { recursive: true })
        writeFileSync(join(root, rel), body)
      },
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

/** Vai pela barra de consoles até a aba (setas, como a pessoa faria). */
export async function goTab(page: Page, slug: string) {
  const tabs = await page.locator('nav.tabs [data-tab]').evaluateAll((els) => els.map((e) => e.getAttribute('data-tab')))
  const target = tabs.indexOf(slug)
  if (target < 0) throw new Error(`aba ${slug} não existe: ${tabs.join(', ')}`)
  for (let i = 0; i < 30; i++) {
    const current = await page.locator('nav.tabs [aria-current="page"]').getAttribute('data-tab')
    if (current === slug) return
    if (!(await page.locator('nav.tabs [data-tab]:focus').count())) await page.locator(`nav.tabs [data-tab="${current}"]`).focus()
    await page.keyboard.press(tabs.indexOf(current) < target ? 'ArrowRight' : 'ArrowLeft')
  }
  throw new Error(`não chegou na aba ${slug}`)
}

/** Abre a tela "Você" (Start no controle, M no teclado). */
export async function you(page: Page) {
  await page.keyboard.press('m')
  await expect(page.locator('main.you')).toBeVisible()
}

/** Abre um item da tela "Você" pelo nome ("Biblioteca", "Estatísticas"...). */
export async function openItem(page: Page, label: string) {
  await you(page)
  await page.locator('main.you [data-item]').filter({ has: page.getByText(label, { exact: true }) }).focus()
  await page.keyboard.press('Enter') // pelo controle: o mouse parado em cima roubaria o foco
}

/** Abre as configurações (pela tela "Você"); depois, as abas são as de sempre. */
export async function settings(page: Page) {
  await openItem(page, 'Biblioteca')
  await expect(page.getByRole('heading', { name: 'Configurações' })).toBeVisible()
}

/** Vai pra "Quem vai jogar?" (avatar › Você › Trocar perfil). */
export async function switchProfile(page: Page) {
  await page.getByRole('button', { name: /^Você: / }).click()
  await page.getByRole('button', { name: 'Trocar perfil' }).click()
}

/** Abre o menu Opções da página do jogo e escolhe um item ("Listas", "Editar informações"...). */
export async function gameOption(page: Page, label: string | RegExp) {
  await page.getByRole('button', { name: 'Opções', exact: true }).click()
  const menu = page.getByRole('dialog', { name: 'Opções' })
  await expect(menu).toBeVisible()
  await menu.getByRole('button', { name: label }).click()
}
