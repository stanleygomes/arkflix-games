// Cores: paleta do console em foco vira variáveis CSS, e o contraste (WCAG)
// é calculado aqui pra testes e pro modo de alto contraste.
import type { Palette } from './types'

/** Paleta da tela inicial (nenhum console em foco). */
export const homePalette: Palette = {
  bg: '#120f1c',
  surface: '#221c33',
  accent: '#ff9b54',
  accent2: '#5ad1ff',
  text: '#f4f1e8',
  muted: '#b7b0c4',
}

/** Alto contraste: preto, branco e amarelo, igual em todo console. */
export const highContrastPalette: Palette = {
  bg: '#000000',
  surface: '#141414',
  accent: '#ffd400',
  accent2: '#7fdcff',
  text: '#ffffff',
  muted: '#e6e6e6',
}

export function hexToRgb(hex: string): [number, number, number] {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map((c) => c + c).join('') : h
  const n = Number.parseInt(full, 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

function channel(c: number): number {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

export function luminance(hex: string): number {
  const [r, g, b] = hexToRgb(hex)
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

/** Razão de contraste WCAG (1 a 21). */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

/** Cor de texto legível sobre o fundo dado (preto ou o creme da paleta). */
export function onColor(bg: string, light = '#f4f1e8'): string {
  return contrast(bg, '#111111') >= contrast(bg, light) ? '#111111' : light
}

export function applyPalette(p: Palette, el: HTMLElement = document.documentElement): void {
  el.style.setProperty('--bg', p.bg)
  el.style.setProperty('--surface', p.surface)
  el.style.setProperty('--accent', p.accent)
  el.style.setProperty('--accent-2', p.accent2)
  el.style.setProperty('--text', p.text)
  el.style.setProperty('--muted', p.muted)
  el.style.setProperty('--on-accent', onColor(p.accent))
}
