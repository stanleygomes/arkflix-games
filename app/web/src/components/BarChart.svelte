<script lang="ts">
  // Gráfico de barras de uma série só (cor do destaque), com valor escrito em
  // cada barra e alternativa em tabela. horizontal = ranking; vertical = tempo.
  type Item = { label: string; value: number; display: string; detail?: string }
  let {
    title,
    items,
    vertical = false,
    every = 1,
  }: { title: string; items: Item[]; vertical?: boolean; every?: number } = $props()
  let table = $state(false)
  const max = $derived(Math.max(1, ...items.map((i) => i.value)))
  const id = `chart-${Math.random().toString(36).slice(2, 8)}`
</script>

<section class="chart" aria-labelledby={id}>
  <header>
    <h2 id={id}>{title}</h2>
    <button class="btn small" data-nav aria-pressed={table} onclick={() => (table = !table)}>{table ? 'Ver gráfico' : 'Ver tabela'}</button>
  </header>
  {#if table}
    <table>
      <caption class="sr-only">{title}</caption>
      <tbody>
        {#each items as it, i (i)}
          <tr><th scope="row">{it.label}</th><td>{it.display}</td>{#if it.detail}<td>{it.detail}</td>{/if}</tr>
        {/each}
      </tbody>
    </table>
  {:else if vertical}
    <ol class="columns" aria-label={title}>
      {#each items as it, i (i)}
        <li title={`${it.label}: ${it.display}${it.detail ? ` · ${it.detail}` : ''}`}>
          <span class="col" style:height={`${(it.value / max) * 100}%`}></span>
          <span class="tick" class:hidden={i % every !== 0 && i !== items.length - 1}>{it.label}</span>
          <span class="sr-only">{it.label}: {it.display}</span>
        </li>
      {/each}
    </ol>
  {:else}
    <ol class="rows" aria-label={title}>
      {#each items as it, i (i)}
        <li title={it.detail ? `${it.label}: ${it.detail}` : undefined}>
          <span class="name">{it.label}</span>
          <span class="track"><span class="bar" style:width={`${Math.max(2, (it.value / max) * 100)}%`}></span></span>
          <span class="val">{it.display}</span>
        </li>
      {/each}
    </ol>
  {/if}
</section>

<style>
  .chart {
    display: flex;
    flex-direction: column;
    gap: 0.8rem;
  }
  header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }
  h2 {
    font-size: 1.15rem;
  }
  .rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.45rem;
  }
  .rows li {
    display: grid;
    grid-template-columns: minmax(7rem, 12rem) minmax(8rem, 1fr) 6.5rem;
    gap: 0.8rem;
    align-items: center;
  }
  .name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .track {
    height: 0.9rem;
    background: color-mix(in srgb, var(--text) 8%, transparent);
  }
  .bar {
    display: block;
    height: 100%;
    background: var(--accent);
    box-shadow: 2px 2px 0 var(--shadow);
  }
  .val {
    text-align: right;
    color: var(--muted);
    font-variant-numeric: tabular-nums;
  }
  .columns {
    list-style: none;
    margin: 0;
    padding: 0;
    height: 9rem;
    display: flex;
    align-items: flex-end;
    gap: 2px;
    border-bottom: 2px solid color-mix(in srgb, var(--text) 25%, transparent);
    padding-bottom: 1.4rem;
    position: relative;
  }
  .columns li {
    flex: 1;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    position: relative;
  }
  .col {
    display: block;
    background: var(--accent);
    min-height: 1px;
  }
  .tick {
    position: absolute;
    bottom: -1.3rem;
    left: 50%;
    transform: translateX(-50%);
    font-size: 0.7rem;
    color: var(--muted);
    white-space: nowrap;
  }
  .tick.hidden {
    visibility: hidden;
  }
  table {
    border-collapse: collapse;
    width: 100%;
  }
  th,
  td {
    text-align: left;
    padding: 0.35rem 0.6rem;
    border-bottom: 1px solid color-mix(in srgb, var(--text) 15%, transparent);
  }
  th {
    font-weight: 400;
  }
</style>
