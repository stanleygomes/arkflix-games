import { describe, expect, it } from 'vitest'
import { choose, neighbor, newGame, rng, safeMoves, step, stuck, type Snake } from '../../src/lib/snake'

const at = (w: number, x: number, y: number) => y * w + x

describe('cobrinha', () => {
  it('nunca escolhe movimento que mata quando há saída', () => {
    // cabeça no canto de cima, comida pra cima-esquerda num beco: o caminho
    // reto bate na parede; a IA tem que ir pra baixo
    const w = 5
    const s: Snake = {
      w,
      h: 5,
      body: [at(w, 0, 0), at(w, 1, 0), at(w, 2, 0)],
      food: at(w, 4, 4),
      score: 0,
      alive: true,
      hunger: 0,
    }
    expect(safeMoves(s)).toEqual(['down'])
    expect(choose(s)).toBe('down')
  })

  it('não entra num bolsão fechado só porque a comida está lá', () => {
    // corpo forma uma parede com uma abertura de 1 casa; a comida está dentro
    // de um bolsão de 1 casa: comer prende a cobra sem alcançar o rabo
    const w = 4
    const body = [at(w, 1, 1), at(w, 1, 2), at(w, 2, 2), at(w, 3, 2), at(w, 3, 1), at(w, 3, 0), at(w, 2, 0)]
    const s: Snake = { w, h: 4, body, food: at(w, 2, 1), score: 0, alive: true, hunger: 0 }
    const d = choose(s)
    expect(d).not.toBeNull()
    expect(safeMoves(s)).toContain(d)
  })

  it('morre só quando não existe nenhuma saída', () => {
    const w = 3
    // cabeça no canto, cercada pelo próprio corpo (rabo longe)
    const body = [at(w, 0, 0), at(w, 1, 0), at(w, 1, 1), at(w, 0, 1), at(w, 0, 2), at(w, 1, 2)]
    const s: Snake = { w, h: 3, body, food: at(w, 2, 2), score: 0, alive: true, hunger: 0 }
    expect(safeMoves(s)).toEqual([])
    expect(step(s, rng(1)).alive).toBe(false)
  })

  it('joga sozinha por muito tempo sem travar e sem suicídio', () => {
    for (let seed = 1; seed <= 6; seed++) {
      const rnd = rng(seed)
      let s = newGame(16, 12, rnd)
      const bad: string[] = []
      for (let i = 0; i < 6000 && s.alive && !stuck(s); i++) {
        const moves = safeMoves(s)
        const d = choose(s)
        if (moves.length ? !d || !moves.includes(d) : d !== null) bad.push(`passo ${i}: ${d} fora de ${moves}`)
        const before = s.body.length
        s = step(s, rnd)
        if (s.alive && (new Set(s.body).size !== s.body.length || s.body.length - before > 1 || neighbor(16, 12, s.body[1], d!) !== s.body[0])) {
          bad.push(`passo ${i}: corpo inválido`)
        }
      }
      expect(bad, `semente ${seed}`).toEqual([])
      expect(s.alive || s.food < 0, `semente ${seed}: morreu`).toBe(true)
      expect(s.score, `semente ${seed}`).toBeGreaterThan(150) // tabuleiro de 192 casas
    }
  }, 30_000)
})
