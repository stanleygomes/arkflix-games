<script lang="ts">
  import { play } from '../../lib/sound'
  import { app } from '../../lib/state.svelte'
  import type { Accessibility } from '../../lib/types'
  import { saveSoon } from './save.svelte'

  const a = $derived(app.settings!.a11y)
  const scales = [1, 1.25, 1.5, 1.75]

  function set<K extends keyof Accessibility>(key: K, value: Accessibility[K]) {
    saveSoon({ ...app.settings!, a11y: { ...a, [key]: value } })
    play('toggle')
  }

  const switches: { key: keyof Accessibility; label: string; help: string }[] = [
    { key: 'highContrast', label: 'Alto contraste', help: 'Preto, branco e amarelo em toda a interface; ignora a cor dos consoles.' },
    { key: 'reduceMotion', label: 'Reduzir animações', help: 'Sem transições, cursor piscando ou rolagem suave.' },
    { key: 'scanlines', label: 'Linhas de TV (CRT)', help: 'Efeito sutil de tela antiga por cima da interface.' },
    { key: 'pixelTitles', label: 'Títulos em fonte pixel', help: 'Desligado, todos os textos usam a fonte de leitura fácil.' },
    { key: 'sounds', label: 'Sons de navegação', help: 'Bipes 16-bit ao mover, confirmar e voltar.' },
    { key: 'music', label: 'Música de fundo', help: 'Trilha chiptune baixa nos menus; para durante o jogo.' },
    { key: 'videos', label: 'Vídeo dos jogos', help: 'No Início, o vídeo do jogo toca no fundo depois de 2 s. Nunca com "Reduzir animações".' },
    { key: 'videoSound', label: 'Som dos vídeos', help: 'Desligado, os vídeos tocam mudos.' },
  ]
</script>

<div class="panel-body">
  <h2>Acessibilidade</h2>

  <fieldset>
    <legend>Tamanho do texto</legend>
    <div class="segmented" role="group" aria-label="Tamanho do texto">
      {#each scales as s (s)}
        <button class="btn" data-nav aria-pressed={a.fontScale === s} onclick={() => set('fontScale', s)}>{Math.round(s * 100)}%</button>
      {/each}
    </div>
  </fieldset>

  <ul class="switches">
    {#each switches as sw (sw.key)}
      <li>
        <button
          class="switch"
          data-nav
          role="switch"
          aria-checked={a[sw.key] as boolean}
          aria-describedby={`help-${sw.key}`}
          onclick={() => set(sw.key, !a[sw.key])}
        >
          <span class="knob" aria-hidden="true"></span>
          <span class="label">{sw.label}</span>
        </button>
        <p class="muted help" id={`help-${sw.key}`}>{sw.help}</p>
      </li>
    {/each}
  </ul>

  {#each [{ key: 'soundVolume' as const, label: 'Volume dos sons' }, { key: 'musicVolume' as const, label: 'Volume da música' }] as v (v.key)}
    <fieldset>
      <legend>{v.label}</legend>
      <div class="stepper">
        <button class="btn small" data-nav aria-label={`Diminuir ${v.label.toLowerCase()}`} onclick={() => set(v.key, Math.max(0, a[v.key] - 10))}>−</button>
        <span class="meter" role="meter" aria-label={v.label} aria-valuenow={a[v.key]} aria-valuemin="0" aria-valuemax="100">
          {#each Array(10) as _, i (i)}<span class="bar" class:on={a[v.key] > i * 10}></span>{/each}
        </span>
        <span class="value pixel">{a[v.key]}</span>
        <button class="btn small" data-nav aria-label={`Aumentar ${v.label.toLowerCase()}`} onclick={() => set(v.key, Math.min(100, a[v.key] + 10))}>+</button>
      </div>
    </fieldset>
  {/each}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.4rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  fieldset {
    border: none;
    padding: 0;
    margin: 0;
  }
  legend {
    font-weight: 700;
    margin-bottom: 0.6rem;
  }
  .segmented,
  .stepper {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .switches {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 1rem;
  }
  .switch {
    font: inherit;
    font-weight: 700;
    color: var(--text);
    background: none;
    border: none;
    display: flex;
    gap: 0.9rem;
    align-items: center;
    padding: 0.3rem;
    cursor: pointer;
  }
  .knob {
    width: 3.2rem;
    height: 1.6rem;
    border: 3px solid color-mix(in srgb, var(--text) 50%, transparent);
    background: color-mix(in srgb, var(--bg) 70%, #000);
    position: relative;
    flex: none;
  }
  .knob::after {
    content: '';
    position: absolute;
    top: 2px;
    left: 2px;
    width: 1rem;
    height: calc(100% - 4px);
    background: var(--muted);
    transition: left var(--t) var(--ease);
  }
  .switch[aria-checked='true'] .knob {
    background: color-mix(in srgb, var(--accent) 40%, #000);
    border-color: var(--accent);
  }
  .switch[aria-checked='true'] .knob::after {
    left: calc(100% - 1rem - 2px);
    background: var(--accent);
  }
  .help {
    margin-left: 4.4rem;
    font-size: 0.9rem;
  }
  .meter {
    display: inline-flex;
    gap: 3px;
  }
  .bar {
    width: 0.7rem;
    height: 1.2rem;
    background: color-mix(in srgb, var(--text) 18%, transparent);
  }
  .bar.on {
    background: var(--accent);
  }
  .value {
    min-width: 2.5em;
    text-align: right;
  }
</style>
