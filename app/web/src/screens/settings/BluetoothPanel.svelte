<script lang="ts">
  // Bluetooth: parear controle sem sair do sofá. Procura, pareia (confia e
  // conecta junto, pra reconectar sozinho), desconecta e esquece. Fora do
  // Linux, só um atalho pras configurações do sistema.
  import Icon from '../../components/Icon.svelte'
  import { api, on } from '../../lib/api'
  import { play } from '../../lib/sound'
  import { confirm, toast } from '../../lib/state.svelte'
  import type { BTDevice, BTInfo } from '../../lib/types'

  let info = $state<BTInfo | null>(null)
  let error = $state('')

  $effect(() => {
    api.btStatus().then((i) => (info = i)).catch((err) => (error = (err as Error).message))
    return on<BTInfo>('bluetooth', (i) => (info = i))
  })

  async function run(fn: () => Promise<unknown>, ok = '') {
    try {
      await fn()
      if (ok) toast(ok, 'info', 2500, 'bluetooth')
    } catch (err) {
      play('error')
      toast((err as Error).message, 'error')
    }
  }

  async function forget(d: BTDevice) {
    if (!(await confirm(`Esquecer ${d.name}? Pra usar de novo, vai ter que parear outra vez.`, 'Esquecer', true))) return
    run(() => api.btForget(d.address))
  }

  const controllers = $derived(info?.devices.filter((d) => d.gamepad) ?? [])
  const others = $derived(info?.devices.filter((d) => !d.gamepad && d.paired) ?? [])
</script>

<div class="panel-body">
  <header class="head">
    <h2>Bluetooth</h2>
    {#if info?.supported && info.powered}
      <button class="btn primary" data-nav data-nav-default disabled={info.scanning} onclick={() => run(() => api.btScan(30))}>
        <Icon name="search" /> {info.scanning ? 'Procurando…' : 'Procurar controles'}
      </button>
    {/if}
  </header>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {:else if !info}
    <p class="muted" aria-busy="true">Olhando o Bluetooth…</p>
  {:else if !info.supported}
    <p class="muted">{info.reason}</p>
    <button class="btn" data-nav data-nav-default onclick={() => run(api.btSettings)}><Icon name="bluetooth" /> Abrir o Bluetooth do sistema</button>
  {:else}
    {#if !info.powered}
      <p class="muted">O Bluetooth está desligado.</p>
      <button class="btn primary" data-nav data-nav-default onclick={() => run(() => api.btPower(true))}><Icon name="power" /> Ligar o Bluetooth</button>
    {:else}
      <p class="muted small">
        Deixe o controle em modo de pareamento (Xbox: botão de parear no topo; PlayStation: Create + PS; 8BitDo/Switch: o botão de
        parear) e procure.
      </p>
      <section aria-labelledby="bt-pads">
        <h3 id="bt-pads">Controles</h3>
        {#if controllers.length === 0}
          <p class="muted">{info.scanning ? 'Procurando… ponha o controle em modo de pareamento.' : 'Nenhum controle por perto.'}</p>
        {:else}
          <ul class="devs">
            {#each controllers as d (d.address)}
              <li class:on={d.connected}>
                <Icon name="pad" />
                <span class="text">
                  <span class="label">{d.name}</span>
                  <span class="muted small">
                    {d.connected ? 'conectado' : d.paired ? 'pareado' : 'novo'}{d.battery >= 0 ? ` · bateria ${d.battery}%` : ''}
                  </span>
                </span>
                <span class="acts">
                  {#if !d.paired}
                    <button class="btn small primary" data-nav onclick={() => run(() => api.btPair(d.address), `Pareando ${d.name}…`)}>Parear</button>
                  {:else if d.connected}
                    <button class="btn small" data-nav onclick={() => run(() => api.btDisconnect(d.address))}>Desconectar</button>
                  {:else}
                    <button class="btn small" data-nav onclick={() => run(() => api.btConnect(d.address))}>Conectar</button>
                  {/if}
                  {#if d.paired}
                    <button class="btn small danger" data-nav onclick={() => forget(d)}>Esquecer</button>
                  {/if}
                </span>
              </li>
            {/each}
          </ul>
        {/if}
      </section>
      {#if others.length}
        <section aria-labelledby="bt-others">
          <h3 id="bt-others">Outros aparelhos pareados</h3>
          <ul class="devs">
            {#each others as d (d.address)}
              <li class:on={d.connected}>
                <Icon name="bluetooth" />
                <span class="text"><span class="label">{d.name}</span><span class="muted small">{d.connected ? 'conectado' : 'pareado'}</span></span>
              </li>
            {/each}
          </ul>
        </section>
      {/if}
      <div class="actions">
        <button class="btn" data-nav onclick={() => run(() => api.btPower(false))}><Icon name="power" /> Desligar o Bluetooth</button>
        <button class="btn" data-nav onclick={() => run(api.btSettings)}>Bluetooth do sistema</button>
      </div>
    {/if}
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.2rem;
  }
  .head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  h3 {
    font-size: 1rem;
    margin-bottom: 0.6rem;
  }
  .devs {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
  }
  .devs li {
    display: grid;
    grid-template-columns: auto 1fr auto;
    gap: 0.9rem;
    align-items: center;
    padding: 0.8rem 1rem;
    border: 2px solid color-mix(in srgb, var(--accent-2) 40%, transparent);
  }
  .devs li.on {
    border-color: color-mix(in srgb, #5dc44f 70%, transparent);
  }
  .text {
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
  }
  .label {
    font-weight: 700;
  }
  .small {
    font-size: 0.85rem;
  }
  .acts,
  .actions {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
</style>
