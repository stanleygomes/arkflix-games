// Cobrinha que joga sozinha (protetor de tela). Pura: estado entra, estado sai.
// IA: vai pela menor rota até a comida só se, depois de comer, ainda alcança
// o próprio rabo; senão segue o rabo pelo caminho mais longo; em último caso,
// o lado com mais espaço. Nunca escolhe uma casa que mata se houver outra.

export type Dir = 'up' | 'down' | 'left' | 'right'

export interface Snake {
  w: number
  h: number
  body: number[] // índices (y*w+x), cabeça primeiro
  food: number
  score: number
  alive: boolean
  hunger: number // passos desde a última comida
}

const DIRS: Record<Dir, [number, number]> = { up: [0, -1], down: [0, 1], left: [-1, 0], right: [1, 0] }
const ORDER: Dir[] = ['up', 'right', 'down', 'left']

/** Gerador determinístico (testes e replays). */
export function rng(seed: number): () => number {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
}

export function newGame(w: number, h: number, rnd: () => number): Snake {
  const y = Math.floor(h / 2)
  const x = Math.floor(w / 3)
  const body = [y * w + x, y * w + x - 1, y * w + x - 2]
  const s: Snake = { w, h, body, food: -1, score: 0, alive: true, hunger: 0 }
  s.food = placeFood(s, rnd)
  return s
}

function placeFood(s: Snake, rnd: () => number): number {
  const taken = new Set(s.body)
  const free: number[] = []
  for (let i = 0; i < s.w * s.h; i++) if (!taken.has(i)) free.push(i)
  return free.length ? free[Math.floor(rnd() * free.length)] : -1
}

/** Casa vizinha na direção, ou -1 se sai do tabuleiro. */
export function neighbor(w: number, h: number, cell: number, d: Dir): number {
  const x = (cell % w) + DIRS[d][0]
  const y = Math.floor(cell / w) + DIRS[d][1]
  return x < 0 || y < 0 || x >= w || y >= h ? -1 : y * w + x
}

/** Corpo depois de andar uma casa (cresce se comer). */
function advance(body: number[], next: number, food: number): number[] {
  const out = [next, ...body]
  if (next !== food) out.pop()
  return out
}

/** Movimentos que não matam agora: dentro do tabuleiro e fora do corpo
 * (o rabo sai do lugar, a menos que a cobra coma nesse passo). */
export function safeMoves(s: Snake): Dir[] {
  const out: Dir[] = []
  for (const d of ORDER) {
    const n = neighbor(s.w, s.h, s.body[0], d)
    if (n < 0) continue
    const grows = n === s.food
    const blocked = grows ? s.body : s.body.slice(0, -1)
    if (!blocked.includes(n)) out.push(d)
  }
  return out
}

/** Menor caminho (lista de casas) de from até to, desviando de blocked. */
function bfs(w: number, h: number, from: number, to: number, blocked: Set<number>): number[] | null {
  const prev = new Map<number, number>([[from, -1]])
  const queue = [from]
  for (let i = 0; i < queue.length; i++) {
    const cur = queue[i]
    if (cur === to) {
      const path: number[] = []
      for (let c = to; c !== from; c = prev.get(c)!) path.unshift(c)
      return path
    }
    for (const d of ORDER) {
      const n = neighbor(w, h, cur, d)
      if (n < 0 || prev.has(n) || (blocked.has(n) && n !== to)) continue
      prev.set(n, cur)
      queue.push(n)
    }
  }
  return null
}

function area(w: number, h: number, from: number, blocked: Set<number>): number {
  const seen = new Set([from])
  const queue = [from]
  for (let i = 0; i < queue.length; i++) {
    for (const d of ORDER) {
      const n = neighbor(w, h, queue[i], d)
      if (n >= 0 && !seen.has(n) && !blocked.has(n)) {
        seen.add(n)
        queue.push(n)
      }
    }
  }
  return seen.size
}

/** A cabeça consegue chegar no rabo? (rabo = saída que sempre se abre) */
function tailReachable(w: number, h: number, body: number[]): boolean {
  if (body.length < 3) return true
  const tail = body[body.length - 1]
  return bfs(w, h, body[0], tail, new Set(body.slice(1, -1))) !== null
}

function dirTo(w: number, from: number, to: number): Dir {
  const dx = (to % w) - (from % w)
  if (dx === 1) return 'right'
  if (dx === -1) return 'left'
  return to > from ? 'down' : 'up'
}

/** Próximo movimento da IA; null = não há movimento que não mate. */
export function choose(s: Snake): Dir | null {
  const moves = safeMoves(s)
  if (moves.length === 0) return null
  const { w, h, body, food } = s

  // 1) rota até a comida, se depois de comer ainda dá pra seguir o rabo
  if (food >= 0) {
    const path = bfs(w, h, body[0], food, new Set(body.slice(0, -1)))
    if (path) {
      let virt = body
      for (const c of path) virt = advance(virt, c, food)
      const d = dirTo(w, body[0], path[0])
      if (moves.includes(d) && (virt.length >= w * h || tailReachable(w, h, virt))) return d
    }
  }

  // 2) segue o rabo pelo caminho mais longo (ganha tempo até abrir rota)
  let best: Dir | null = null
  let bestScore = -1
  for (const d of moves) {
    const next = advance(body, neighbor(w, h, body[0], d), food)
    if (!tailReachable(w, h, next)) continue
    const tail = next[next.length - 1]
    const dist = bfs(w, h, next[0], tail, new Set(next.slice(1, -1)))?.length ?? 0
    if (dist > bestScore) {
      bestScore = dist
      best = d
    }
  }
  if (best) return best

  // 3) sem rabo à vista: o lado com mais espaço livre
  for (const d of moves) {
    const next = advance(body, neighbor(w, h, body[0], d), food)
    const room = area(w, h, next[0], new Set(next.slice(1)))
    if (room > bestScore) {
      bestScore = room
      best = d
    }
  }
  return best
}

/** Um passo do jogo. Não muda o estado recebido. */
export function step(s: Snake, rnd: () => number): Snake {
  if (!s.alive) return s
  const d = choose(s)
  if (!d) return { ...s, alive: false }
  const next = neighbor(s.w, s.h, s.body[0], d)
  const ate = next === s.food
  const out: Snake = { ...s, body: advance(s.body, next, s.food), hunger: ate ? 0 : s.hunger + 1 }
  if (ate) {
    out.score = s.score + 1
    out.food = placeFood(out, rnd)
    if (out.food < 0) out.alive = false // tabuleiro cheio: ganhou
  }
  return out
}

/** Presa num ciclo sem comer? (o protetor reinicia) */
export function stuck(s: Snake): boolean {
  return s.hunger > s.w * s.h * 3
}
