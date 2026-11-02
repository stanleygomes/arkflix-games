import { describe, expect, it } from 'vitest'
import { changes, familyOf, padName, snapshot, startBrowserPads } from '../../src/lib/browserpads'
import type { PadEvent } from '../../src/lib/types'

const pad = (pressed: number[] = [], axes = [0, 0, 0, 0], id = 'Xbox Wireless Controller (STANDARD GAMEPAD Vendor: 045e Product: 0b13)') => ({
  index: 0,
  id,
  connected: true,
  buttons: Array.from({ length: 17 }, (_, i) => ({ pressed: pressed.includes(i), value: pressed.includes(i) ? 1 : 0 })),
  axes,
})

describe('controle pela Gamepad API (Windows e macOS)', () => {
  it('reconhece o fabricante e limpa o nome', () => {
    expect(familyOf('Xbox Wireless Controller (STANDARD GAMEPAD Vendor: 045e)')).toBe('xbox')
    expect(familyOf('DualSense Wireless Controller (Vendor: 054c Product: 0ce6)')).toBe('playstation')
    expect(familyOf('8BitDo Ultimate 2C (Vendor: 2dc8)')).toBe('8bitdo')
    expect(familyOf('Pro Controller (Vendor: 057e)')).toBe('nintendo')
    expect(familyOf('USB Gamepad')).toBe('generic')
    expect(padName('Xbox Wireless Controller (STANDARD GAMEPAD Vendor: 045e)')).toBe('Xbox Wireless Controller')
  })

  it('vira os mesmos eventos do Go: botão e analógico', () => {
    const p = { index: 0, name: 'Xbox', family: 'xbox' as const }
    const ev = changes(p, snapshot(pad()), snapshot(pad([0, 13], [0.9, 0, 0, 0])))
    expect(ev).toEqual([
      { pad: 0, name: 'Xbox', family: 'xbox', button: 'south', pressed: true },
      { pad: 0, name: 'Xbox', family: 'xbox', button: 'down', pressed: true },
      { pad: 0, name: 'Xbox', family: 'xbox', button: 'right', pressed: true, source: 'stick' },
    ])
    expect(changes(p, snapshot(pad([0])), snapshot(pad([0])))).toEqual([]) // segurar não repete
  })

  it('avisa quem entrou e sai, e lê a cada quadro', async () => {
    let state: ReturnType<typeof pad> | null = pad()
    const events: PadEvent[] = []
    const lists: number[] = []
    const stop = startBrowserPads((e) => events.push(e), (p) => lists.push(p.length), () => [state])
    await new Promise((r) => setTimeout(r, 40))
    state = pad([9])
    await new Promise((r) => setTimeout(r, 40))
    state = null
    await new Promise((r) => setTimeout(r, 40))
    stop()
    expect(events.map((e) => `${e.button}:${e.pressed}`)).toEqual(['start:true'])
    expect(lists).toEqual([1, 0])
  })
})
