<script lang="ts">
  // Conquistas (RetroAchievements), por perfil: entra uma vez e os jogos do
  // RetroArch já contam troféus. A senha só serve pro login; fica um token.
  import Icon from '../../components/Icon.svelte'
  import { api } from '../../lib/api'
  import { dateTime } from '../../lib/format'
  import { play } from '../../lib/sound'
  import { confirm, isKids, prompt, toast } from '../../lib/state.svelte'
  import type { CheevosInfo } from '../../lib/types'

  let info = $state<CheevosInfo | null>(null)
  let user = $state('')

  $effect(() => {
    api.cheevosStatus().then((i) => (info = i)).catch(() => {})
  })

  async function login() {
    let name = ''
    const typed = await prompt('Usuário do RetroAchievements', user, 'Continuar', async (v) => {
      if (!v.trim()) return 'Digite o usuário'
      name = v.trim()
      return null
    }, { max: 40 })
    if (typed === null) return
    user = name
    await prompt('Senha do RetroAchievements', '', 'Entrar', async (pass) => {
      if (!pass) return 'Digite a senha'
      try {
        info = await api.cheevosLogin(user, pass)
        return null
      } catch (err) {
        return (err as Error).message
      }
    }, { max: 200, secret: true })
    if (info?.connected) {
      play('accept')
      toast(`Conquistas ligadas como ${info.user}`, 'ok')
    }
  }

  async function logout() {
    if (!(await confirm('Sair do RetroAchievements neste perfil? As conquistas já ganhas ficam na sua conta.', 'Sair', true))) return
    try {
      await api.cheevosLogout()
      info = await api.cheevosStatus()
    } catch (err) {
      toast((err as Error).message, 'error')
    }
  }
</script>

<div class="panel-body">
  <h2>Conquistas</h2>
  {#if info?.connected}
    <p class="status"><Icon name="trophy" /> Entrou como <strong>{info.user}</strong>{info.since ? ` desde ${dateTime(info.since)}` : ''}.</p>
    <ul class="facts">
      <li>Os jogos do RetroArch (NES, SNES, Mega Drive, N64, PS1, arcade...) contam conquistas sozinhos, com aviso na tela.</li>
      <li>Na página de cada jogo aparece o que você já ganhou e o que falta.</li>
      <li class="muted">Modo normal (dá pra salvar e carregar estado). PSP e PS2 não têm conquistas por aqui.</li>
    </ul>
    {#if !isKids()}
      <button class="btn danger" data-nav data-nav-default onclick={logout}>Sair do RetroAchievements</button>
    {/if}
  {:else if info}
    <p class="muted">
      Ganhe troféus nos clássicos com uma conta grátis do <strong>retroachievements.org</strong>. Cada perfil usa a sua. A senha só é usada
      pra entrar: o app guarda um token.
    </p>
    {#if isKids()}
      <p class="muted">Peça pra um adulto ligar as conquistas neste perfil.</p>
    {:else}
      <button class="btn primary" data-nav data-nav-default onclick={login}><Icon name="trophy" /> Entrar no RetroAchievements</button>
    {/if}
  {/if}
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1.1rem;
    align-items: flex-start;
  }
  h2 {
    font-size: 1.5rem;
  }
  .status {
    display: flex;
    gap: 0.6rem;
    align-items: center;
    flex-wrap: wrap;
  }
  .facts {
    display: grid;
    gap: 0.4rem;
    padding-left: 1.2rem;
    margin: 0;
  }
</style>
