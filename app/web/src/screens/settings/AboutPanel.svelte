<script lang="ts">
  import Glyph from '../../components/Glyph.svelte'
  import { app, glyphFamily } from '../../lib/state.svelte'

  const family = $derived(glyphFamily())
</script>

<div class="panel-body">
  <h2>Sobre</h2>
  <dl class="facts">
    <dt>Versão</dt>
    <dd class="pixel">{app.version}</dd>
    <dt>Modo</dt>
    <dd>{app.mode === 'desktop' ? 'App desktop' : 'Navegador'}</dd>
    <dt>Jogos</dt>
    <dd>{app.games.length} em {app.systems.filter((s) => app.games.some((g) => g.system === s.slug)).length} consoles</dd>
  </dl>

  <h3>Na interface</h3>
  <ul class="keys">
    <li><Glyph button="south" {family} /> abrir / confirmar</li>
    <li><Glyph button="east" {family} /> voltar</li>
    <li><Glyph button="north" {family} /> favorito (ou apagar o ponto salvo em foco)</li>
    <li><Glyph button="west" {family} /> detalhes do jogo</li>
    <li><Glyph button="l1" {family} /> <Glyph button="r1" {family} /> trocar console / seção</li>
    <li><Glyph button="start" {family} /> configurações</li>
    <li>Teclado: setas, Enter, Esc, F (favorito), I (detalhes), Q/E (seções), F11 (tela cheia)</li>
  </ul>

  <h3>Dentro do jogo (RetroArch)</h3>
  <ul class="keys">
    <li><Glyph button="select" {family} /> + <Glyph button="start" {family} /> sair · Esc no teclado</li>
    <li><Glyph button="select" {family} /> + <Glyph button="r1" {family} /> salvar · F2</li>
    <li><Glyph button="select" {family} /> + <Glyph button="l1" {family} /> carregar · F4</li>
    <li><Glyph button="select" {family} /> + <Glyph button="north" {family} /> menu · F1</li>
  </ul>

  <h3>Créditos</h3>
  <p class="muted">
    Fontes Pixelify Sans e Atkinson Hyperlegible (SIL Open Font License). Emuladores: RetroArch, PPSSPP e PCSX2, cada um com a
    sua licença (veja o README). Sons e música sintetizados pelo próprio app.
  </p>
</div>

<style>
  .panel-body {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  h2 {
    font-size: 1.5rem;
  }
  h3 {
    font-size: 1.1rem;
    margin-top: 0.5rem;
  }
  .facts {
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 0.4rem 1.2rem;
    margin: 0;
  }
  dt {
    color: var(--muted);
  }
  dd {
    margin: 0;
  }
  .keys {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: 0.6rem;
  }
  .keys li {
    display: flex;
    gap: 0.4rem;
    align-items: center;
    flex-wrap: wrap;
  }
</style>
