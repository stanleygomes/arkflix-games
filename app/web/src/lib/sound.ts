// Sons 16-bit sintetizados na hora (Web Audio): bipes de menu e uma trilha
// chiptune original em loop. Sem arquivos de áudio, sem licença de terceiros.

let ctx: AudioContext | null = null
let master: GainNode | null = null
let musicGain: GainNode | null = null
let sfxOn = true
let sfxVolume = 0.6
let musicOn = false
let musicVolume = 0.25
let timer: number | null = null
let step = 0
let nextTime = 0

function audio(): AudioContext | null {
  if (typeof AudioContext === 'undefined') return null
  if (!ctx) {
    ctx = new AudioContext()
    master = ctx.createGain()
    master.connect(ctx.destination)
    musicGain = ctx.createGain()
    musicGain.gain.value = musicVolume * 0.35
    musicGain.connect(master)
  }
  return ctx
}

/** O navegador só libera áudio depois de uma interação: chame no primeiro input. */
export function unlock() {
  const a = audio()
  if (a && a.state === 'suspended') a.resume().catch(() => {})
  if (musicOn) startMusic()
}

function tone(freq: number, start: number, dur: number, type: OscillatorType, vol: number, dest: AudioNode) {
  const a = ctx!
  const osc = a.createOscillator()
  const g = a.createGain()
  osc.type = type
  osc.frequency.setValueAtTime(freq, start)
  g.gain.setValueAtTime(0.0001, start)
  g.gain.exponentialRampToValueAtTime(vol, start + 0.005)
  g.gain.exponentialRampToValueAtTime(0.0001, start + dur)
  osc.connect(g)
  g.connect(dest)
  osc.start(start)
  osc.stop(start + dur + 0.02)
}

export type Sfx = 'move' | 'accept' | 'back' | 'error' | 'toggle' | 'start'

const patterns: Record<Sfx, [number, number, number][]> = {
  // [frequência, início (s), duração (s)]
  move: [[988, 0, 0.035]],
  accept: [[660, 0, 0.05], [990, 0.05, 0.08]],
  back: [[523, 0, 0.05], [392, 0.05, 0.08]],
  error: [[180, 0, 0.12], [150, 0.12, 0.14]],
  toggle: [[784, 0, 0.04], [1175, 0.04, 0.05]],
  start: [[523, 0, 0.07], [659, 0.07, 0.07], [784, 0.14, 0.07], [1047, 0.21, 0.18]],
}

export function play(name: Sfx) {
  if (!sfxOn) return
  const a = audio()
  if (!a || a.state !== 'running' || !master) return
  const t = a.currentTime
  for (const [f, s, d] of patterns[name]) tone(f, t + s, d, 'square', 0.12 * sfxVolume, master)
}

// Trilha: 32 passos de colcheia, 112 bpm. Melodia em Lá menor pentatônica.
const BPM = 112
const N = (n: string): number => {
  if (n === '-') return 0
  const names = ['C', 'C#', 'D', 'D#', 'E', 'F', 'F#', 'G', 'G#', 'A', 'A#', 'B']
  const m = /^([A-G]#?)(\d)$/.exec(n)!
  const midi = names.indexOf(m[1]) + 12 * (Number(m[2]) + 1)
  return 440 * 2 ** ((midi - 69) / 12)
}
const lead = 'A4 - C5 E5 D5 - C5 A4 G4 - A4 C5 D5 - E5 - G5 - E5 D5 C5 - D5 E5 A4 - - - G4 A4 C5 - - -'.split(' ').map(N)
const bass = 'A2 - A3 - A2 - A3 - F2 - F3 - F2 - F3 - C3 - C4 - C3 - C4 - G2 - G3 - G2 - E3 -'.split(' ').map(N)

function schedule() {
  const a = ctx
  if (!a || !musicGain) return
  const stepDur = 60 / BPM / 2
  while (nextTime < a.currentTime + 0.25) {
    const i = step % 32
    if (lead[i]) tone(lead[i], nextTime, stepDur * 0.9, 'square', 0.2, musicGain)
    if (bass[i]) tone(bass[i], nextTime, stepDur * 1.6, 'triangle', 0.35, musicGain)
    if (i % 4 === 2) tone(6000 + (i % 8) * 300, nextTime, 0.02, 'square', 0.05, musicGain) // chimbal
    nextTime += stepDur
    step++
  }
}

function startMusic() {
  const a = audio()
  if (!a || a.state !== 'running' || timer !== null) return
  nextTime = a.currentTime + 0.1
  timer = window.setInterval(schedule, 60)
}

function stopMusic() {
  if (timer !== null) {
    clearInterval(timer)
    timer = null
  }
}

export function configure(opts: { sounds: boolean; soundVolume: number; music: boolean; musicVolume: number }) {
  sfxOn = opts.sounds
  sfxVolume = opts.soundVolume / 100
  musicOn = opts.music
  musicVolume = opts.musicVolume / 100
  if (musicGain) musicGain.gain.value = musicVolume * 0.35
  if (musicOn) startMusic()
  else stopMusic()
}

/** Pausa a música enquanto um jogo roda. */
export function pauseMusic(paused: boolean) {
  if (paused) stopMusic()
  else if (musicOn) startMusic()
}
