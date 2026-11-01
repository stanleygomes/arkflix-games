<script lang="ts">
  // Vídeo do jogo no fundo do destaque: entra depois de 2 s parado no mesmo
  // jogo, mudo por padrão. Nunca com "reduzir animações" (do app ou do sistema).
  import { pauseMusic } from '../lib/sound'
  import { app } from '../lib/state.svelte'
  import type { Game } from '../lib/types'

  let { game }: { game: Game | undefined } = $props()

  const DELAY = 2000
  let showing = $state('')
  const a = $derived(app.settings?.a11y)
  const systemReduce = typeof matchMedia === 'function' && matchMedia('(prefers-reduced-motion: reduce)').matches
  const allowed = $derived(!!a?.videos && !a.reduceMotion && !systemReduce && !app.playing)
  const sound = $derived(!!a?.videoSound)

  $effect(() => {
    const id = game?.id
    const src = game?.video
    showing = ''
    if (!id || !src || !allowed) return
    const t = setTimeout(() => (showing = id), DELAY)
    return () => clearTimeout(t)
  })

  // com som, a trilha do menu para enquanto o vídeo toca
  $effect(() => {
    if (!showing || !sound) return
    pauseMusic(true)
    return () => pauseMusic(false)
  })

  function volume(el: HTMLVideoElement) {
    $effect(() => {
      el.muted = !sound
      el.volume = Math.min(1, (a?.soundVolume ?? 60) / 100)
    })
  }
</script>

{#if game?.video && showing === game.id}
  <video class="hero-video" src={game.video} autoplay loop playsinline muted aria-hidden="true" data-game={game.id} use:volume></video>
{/if}

<style>
  .hero-video {
    position: fixed;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    image-rendering: pixelated;
    opacity: 0.4;
    z-index: -1;
    pointer-events: none;
    mask-image: linear-gradient(to bottom, #000 30%, transparent 85%);
    animation: fade 0.6s steps(6);
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
</style>
