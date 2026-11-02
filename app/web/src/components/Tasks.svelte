<script lang="ts">
  // Tarefas longas (baixar capas): barra de progresso enquanto roda, aviso no fim.
  import { on } from '../lib/api'
  import { toast } from '../lib/state.svelte'
  import type { TaskEvent } from '../lib/types'

  let tasks = $state<TaskEvent[]>([])

  $effect(() =>
    on<TaskEvent>('task', (ev) => {
      const i = tasks.findIndex((t) => t.id === ev.id)
      if (ev.state === 'running') {
        if (i >= 0) tasks[i] = ev
        else tasks.push(ev)
        return
      }
      if (i >= 0) tasks.splice(i, 1)
      toast(ev.state === 'done' ? `${ev.label}: ${ev.message}` : `${ev.label} falhou: ${ev.message}`, ev.state === 'done' ? 'ok' : 'error', 7000)
    }),
  )
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
    left: 2rem;
    bottom: 4.5rem;
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
