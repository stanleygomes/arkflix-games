<script lang="ts">
  // Capa do jogo. Sem capa, desenha a mídia do console em pixel (cartucho,
  // disco, UMD, cartão, fliperama) com o título no rótulo.
  import type { Game, System } from '../lib/types'

  let { game, system, large = false }: { game: Game; system?: System; large?: boolean } = $props()
  let failed = $state(false)
  const media = $derived(system?.media ?? 'cartridge')
  const accent = $derived(system?.palette.accent ?? '#ff3cac')
  const accent2 = $derived(system?.palette.accent2 ?? '#5ad1ff')
</script>

<div class="art {media}" class:large style:--a={accent} style:--a2={accent2}>
  {#if game.cover && !failed}
    <img src={game.cover} alt="" loading="lazy" decoding="async" onerror={() => (failed = true)} />
  {:else}
    <div class="shape" aria-hidden="true">
      <span class="label pixel">{game.name}</span>
      <span class="brand pixel">{system?.short ?? ''}</span>
    </div>
  {/if}
</div>

<style>
  .art {
    aspect-ratio: 3 / 4;
    width: 100%;
    position: relative;
    background: color-mix(in srgb, var(--a) 18%, #0b0a10);
    border: 3px solid #05040a;
    box-shadow: inset 0 0 0 3px color-mix(in srgb, var(--a) 45%, transparent);
    overflow: hidden;
  }
  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    display: block;
  }
  .shape {
    position: absolute;
    inset: 10%;
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding: 8%;
    gap: 0.4em;
  }
  .label {
    font-size: 0.8rem;
    line-height: 1.15;
    color: #111;
    background: var(--a);
    padding: 0.4em;
    border: 3px solid #05040a;
    box-shadow: 3px 3px 0 #05040a;
    overflow-wrap: anywhere;
  }
  .large .label {
    font-size: 1.3rem;
  }
  .brand {
    font-size: 0.7rem;
    color: var(--a2);
    letter-spacing: 0.15em;
  }
  /* cartucho: corpo com chanfro em cima e sulcos */
  .cartridge .shape {
    background:
      repeating-linear-gradient(to bottom, transparent 0 6px, rgba(0, 0, 0, 0.25) 6px 9px) top / 100% 18% no-repeat,
      #6d6a75;
    clip-path: polygon(8% 0, 92% 0, 100% 6%, 100% 100%, 0 100%, 0 6%);
  }
  /* disco: círculo com furo e reflexo */
  .disc .shape {
    border-radius: 50%;
    aspect-ratio: 1;
    inset: auto 8% 12% 8%;
    background:
      radial-gradient(circle, #0b0a10 0 9%, #c9ccd6 9.5% 14%, transparent 14.5%),
      conic-gradient(from 30deg, #d8dbe3, #8f93a3, #e7e9ef, #9aa0b2, #d8dbe3);
    justify-content: center;
    align-items: center;
    text-align: center;
  }
  .disc .label {
    font-size: 0.6rem;
    max-width: 60%;
    transform: translateY(85%);
  }
  .umd .shape {
    border-radius: 18% 18% 6% 6%;
    background: radial-gradient(circle at 50% 42%, #c9ccd6 0 24%, #3a3d48 25%);
  }
  .card .shape {
    inset: 20% 16%;
    background: #3c3f48;
    clip-path: polygon(0 0, 84% 0, 100% 10%, 100% 100%, 0 100%);
  }
  .arcade .shape {
    inset: 6%;
    background:
      linear-gradient(var(--a2), var(--a2)) top / 100% 16% no-repeat,
      linear-gradient(#000, #000) 50% 34% / 70% 34% no-repeat,
      #2a1630;
  }
</style>
