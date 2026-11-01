import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))
export const WEB_DIR = resolve(here, '../..')
export const APP_DIR = resolve(WEB_DIR, '..')
export const DIST = resolve(WEB_DIR, 'dist')
export const BIN = resolve(APP_DIR, '..', 'build', 'e2e', 'arkflix-games')
