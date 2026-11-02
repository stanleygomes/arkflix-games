<script lang="ts">
  // Menu do jogo selecionado (Start/Options no Início): o básico e as opções.
  // Listas e configurar abrem no lugar do menu; ao fechar, o foco volta pro card.
  import { tick } from 'svelte'
  import GameConfig from './GameConfig.svelte'
  import GameOptions from './GameOptions.svelte'
  import ListPicker from './ListPicker.svelte'

  let { gameId, onclose }: { gameId: string; onclose: () => void } = $props()
  let mode = $state<'options' | 'lists' | 'config'>('options')
  const from = document.activeElement as HTMLElement | null

  function close() {
    onclose()
    tick().then(() => from?.isConnected && from.focus())
  }
</script>

{#if mode === 'options'}
  <GameOptions {gameId} quick onclose={close} onlists={() => (mode = 'lists')} onconfig={() => (mode = 'config')} />
{:else if mode === 'lists'}
  <ListPicker {gameId} onclose={close} />
{:else}
  <GameConfig {gameId} onclose={close} />
{/if}
