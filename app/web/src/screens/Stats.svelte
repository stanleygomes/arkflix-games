<script lang="ts">
  import BarChart from '../components/BarChart.svelte'
  import HelpBar from '../components/HelpBar.svelte'
  import TopBar from '../components/TopBar.svelte'
  import { api } from '../lib/api'
  import { playTime } from '../lib/format'
  import { applyPalette, highContrastPalette, homePalette } from '../lib/palette'
  import { app, system, toast } from '../lib/state.svelte'
  import type { Stats } from '../lib/types'

  const periods = [
    { id: 'all', label: 'Sempre' },
    { id: '30d', label: '30 dias' },
    { id: '7d', label: '7 dias' },
  ]
  let period = $state('all')
  let data = $state<Stats | null>(null)

  $effect(() => {
    api
      .stats(period)
      .then((s) => (data = s))
      .catch((err) => toast((err as Error).message, 'error'))
  })
  $effect(() => {
    applyPalette(app.settings?.a11y.highContrast ? highContrastPalette : homePalette)
  })

  const weekdays = ['Dom', 'Seg', 'Ter', 'Qua', 'Qui', 'Sex', 'Sáb']
  const dur = (s: number) => (s > 0 ? playTime(s) : '0 min')
  const dayLabel = (d: string) => d.slice(8, 10) + '/' + d.slice(5, 7)
</script>

<div class="page">
  <TopBar actions={false} />
  <main class="body">
    <header class="head">
      <h1 class="pixel">Estatísticas</h1>
      <div class="segmented" role="group" aria-label="Período">
        {#each periods as p, i (p.id)}
          <button class="btn small" data-nav data-nav-default={i === 0 ? '' : undefined} aria-pressed={period === p.id} onclick={() => (period = p.id)}>
            {p.label}
          </button>
        {/each}
      </div>
    </header>
    {#if data}
      <dl class="tiles">
        <div class="tile"><dt>Tempo jogando</dt><dd class="pixel">{dur(data.totalSeconds)}</dd></div>
        <div class="tile"><dt>Partidas</dt><dd class="pixel">{data.plays}</dd></div>
        <div class="tile"><dt>Jogos jogados</dt><dd class="pixel">{data.gamesPlayed} <small>de {data.libraryGames}</small></dd></div>
        <div class="tile">
          <dt>Dia com mais jogo</dt>
          <dd class="pixel">{data.bestDay.seconds ? `${dayLabel(data.bestDay.date)} · ${dur(data.bestDay.seconds)}` : '—'}</dd>
        </div>
      </dl>
      {#if data.sessions === 0}
        <p class="muted empty">Nenhuma partida neste período. Jogue alguma coisa e volte aqui.</p>
      {:else}
        <div class="grid">
          <BarChart
            title="Jogos com mais tempo"
            items={data.byGame.slice(0, 10).map((g) => ({
              label: g.name,
              value: g.seconds,
              display: dur(g.seconds),
              detail: `${g.plays} partida${g.plays === 1 ? '' : 's'} · ${system(g.system)?.short ?? g.system}`,
            }))}
          />
          <BarChart
            title="Tempo por console"
            items={data.bySystem
              .filter((s) => s.games > 0 || s.seconds > 0)
              .map((s) => ({
                label: system(s.system)?.name ?? s.system,
                value: s.seconds,
                display: dur(s.seconds),
                detail: `${s.played} de ${s.games} jogos jogados${s.games ? ` (${Math.round((s.played / s.games) * 100)}%)` : ''}`,
              }))}
          />
          <BarChart
            title="Dia da semana"
            vertical
            items={data.byWeekday.map((v, i) => ({ label: weekdays[i], value: v, display: dur(v) }))}
          />
          <BarChart
            title="Últimos 30 dias"
            vertical
            every={5}
            items={data.byDay.map((d) => ({ label: dayLabel(d.date), value: d.seconds, display: dur(d.seconds) }))}
          />
        </div>
      {/if}
    {/if}
  </main>
  <HelpBar hints={[{ button: 'south', label: 'Escolher' }, { button: 'east', label: 'Voltar' }]} />
</div>

<style>
  .page {
    height: 100vh;
    display: flex;
    flex-direction: column;
  }
  .body {
    flex: 1;
    min-height: 0;
    overflow-y: auto;
    padding: 0.5rem 2.5rem 1.5rem 3.5rem;
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }
  .head {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    flex-wrap: wrap;
  }
  h1 {
    font-size: 2.2rem;
  }
  .segmented {
    display: flex;
    gap: 0.5rem;
  }
  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: 1rem;
    margin: 0;
  }
  .tile {
    background: var(--surface);
    border: 3px solid color-mix(in srgb, var(--text) 18%, var(--surface));
    box-shadow: 4px 4px 0 var(--shadow);
    padding: 0.9rem 1rem;
  }
  dt {
    color: var(--muted);
    font-size: 0.9rem;
  }
  dd {
    margin: 0.3rem 0 0;
    font-size: 1.6rem;
  }
  dd small {
    font-size: 0.9rem;
    color: var(--muted);
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(34rem, 1fr));
    gap: 2.2rem 3.5rem;
  }
  .empty {
    font-size: 1.1rem;
  }
</style>
