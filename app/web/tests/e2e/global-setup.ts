// Compila o binário Go uma vez (modo web, sem janela) e confere a interface compilada.
import { execFileSync } from 'node:child_process'
import { existsSync } from 'node:fs'
import { BIN, DIST, APP_DIR } from './paths'

export default function setup() {
  if (!existsSync(`${DIST}/index.html`)) {
    throw new Error('interface não compilada: rode npm run build antes')
  }
  execFileSync('go', ['build', '-o', BIN, '.'], { cwd: APP_DIR, stdio: 'inherit' })
}
