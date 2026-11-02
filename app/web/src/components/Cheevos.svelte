<script lang="ts">
  // Conquistas do jogo (RetroAchievements): placar e os troféus, ganhos em
  // cor e os que faltam em cinza. Só aparece com conta ligada no perfil.
  import { api } from '../lib/api'
  import type { GameCheevos } from '../lib/types'
  import Icon from './Icon.svelte'

  let { id }: { id: string } = $props()
  let data = $state<GameCheevos | null>(null)
  let error = $state('')

  $effect(() => {
    data = null
    api
      .gameCheevos(id)
      .then((d) => (data = d))
      .catch((err) => (error = (err as Error).message))
  })

  const pct = $derived(data && data.total ? Math.round((data.unlocked / data.total) * 100) : 0)
</script>

{#if error || (data?.connected && (data.supported || data.reason))}
  <section class="cheevos" aria-labelledby="cheevos-title">
    <h2 id="cheevos-title"><Icon name="trophy" /> Conquistas</h2>
    {#if error}
      <p class="muted">{error}</p>
    {:else if data && !data.supported}
      <p class="muted">{data.reason}</p>
    {:else if data}
      <p class="score">
        <span class="hud big">{data.unlocked}/{data.total}</span>
        <span class="muted">· {data.pointsUnlocked} de {data.points} pontos · {pct}%</span>
      </p>
      <div class="energy" role="progressbar" aria-label="Conquistas ganhas" aria-valuemin="0" aria-valuemax="100" aria-valuenow={pct}>
        <span style:width={`${Math.floor(pct / 5) * 5}%`}></span>
      </div>
      <ul class="list">
        {#each data.list ?? [] as c (c.id)}
          <li class:got={c.unlocked}>
            <img src={c.badgeUrl} alt="" loading="lazy" width="48" height="48" />
            <span class="text">
              <span class="title">{c.title}<span class="sr-only">{c.unlocked ? ' (ganha)' : ' (falta)'}</span></span>
              <span class="muted desc">{c.description}</span>
            </span>
            <span class="pts hud">{c.points}</span>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
{/if}

<style>
  .cheevos {
    display: grid;
    gap: 0.7rem;
  }
  h2 {
    display: flex;
    gap: 0.5rem;
    align-items: center;
  }
  .score {
    margin: 0;
  }
  .big {
    font-size: 1.6rem;
    color: #fff;
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(17rem, 1fr));
    gap: 0.5rem;
    overflow-y: auto;
  }
  .list li {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.7rem;
    align-items: center;
    padding: 0.45rem 0.6rem;
    border: 2px solid color-mix(in srgb, var(--accent-2) 30%, transparent);
    opacity: 0.6;
  }
  .list li.got {
    opacity: 1;
    border-color: var(--focus);
  }
  img {
    image-rendering: pixelated;
  }
  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }
  .title {
    font-weight: 700;
  }
  .desc {
    font-size: 0.8rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .pts {
    color: var(--focus);
  }
</style>
