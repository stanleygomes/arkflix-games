<script lang="ts">
  // Grade de jogos (busca, biblioteca, coleções). Confirmar abre o jogo.
  import { go, system } from '../lib/state.svelte'
  import type { Game } from '../lib/types'
  import GameArt from './GameArt.svelte'
  import Icon from './Icon.svelte'

  let { games, label, showSystem = true }: { games: Game[]; label: string; showSystem?: boolean } = $props()
</script>

<ul class="grid" aria-label={label} data-nav-group="grid">
  {#each games as g (g.id)}
    {@const sys = system(g.system)}
    <li>
      <button class="card" data-nav data-game={g.id} onclick={() => go({ name: 'game', id: g.id })}>
        <GameArt game={g} system={sys} />
        <span class="name">{g.name}</span>
        {#if showSystem}<span class="sys">{sys?.short ?? g.system}{g.year ? ` · ${g.year}` : ''}</span>{/if}
        {#if g.favorite}<span class="badge" aria-label="favorito"><Icon name="star" /></span>{/if}
      </button>
    </li>
  {/each}
</ul>

<style>
  .grid {
    list-style: none;
    margin: 0;
    padding: 0.6rem;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(9rem, 1fr));
    gap: 1.4rem 1.1rem;
  }
  .card {
    width: 100%;
    padding: 0;
    border: none;
    background: none;
    color: var(--text);
    font: inherit;
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    cursor: pointer;
  }
  .name {
    font-weight: 700;
    font-size: 0.95rem;
    line-height: 1.2;
    overflow: hidden;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
  }
  .sys {
    font-size: 0.8rem;
    color: var(--muted);
  }
  .badge {
    position: absolute;
    top: -0.5rem;
    right: -0.5rem;
    color: var(--on-accent);
    background: var(--accent);
    border: 2px solid #05040a;
    padding: 0.2rem;
    display: grid;
  }
</style>
