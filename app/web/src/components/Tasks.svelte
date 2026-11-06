<script lang="ts">
  // Tarefas longas em andamento (downloads, capas, instalação): barra de
  // progresso flutuante. O aviso do fim e o histórico ficam na central.
  import { activity } from '../lib/activity.svelte'
  import { app } from '../lib/state.svelte'

  // na própria central a barra seria repetida
  const tasks = $derived(app.stack[app.stack.length - 1].name === 'activity' ? [] : activity.tasks.filter((t) => t.state === 'running').slice(0, 3))
</script>

{#if tasks.length}
  <div class="tasks" aria-live="polite">
    {#each tasks as t (t.id)}
      <div class="task">
        <span class="label">{t.label}…</span>
        <div class="energy" role="progressbar" aria-label={t.label} aria-valuemin="0" aria-valuemax="100" aria-valuenow={Math.round(t.progress * 100)}>
          <span style:width={`${Math.floor(t.progress * 20) * 5}%`}></span>
        </div>
        <span class="pct pixel">{Math.round(t.progress * 100)}%</span>
      </div>
    {/each}
  </div>
{/if}

<style>
  .tasks {
    position: fixed;
    left: calc(2rem + var(--safe));
    bottom: calc(4.5rem + var(--safe));
    z-index: 50;
    display: grid;
    gap: 0.5rem;
  }
  .task {
    display: flex;
    gap: 0.8rem;
    align-items: center;
    background: var(--surface);
    border: 3px solid var(--accent-2);
    box-shadow: var(--glow-2);
    padding: 0.6rem 0.9rem;
  }
  .energy {
    width: 12rem;
  }
</style>
