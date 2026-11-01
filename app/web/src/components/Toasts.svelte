<script lang="ts">
  import { app } from '../lib/state.svelte'
  import Icon from './Icon.svelte'
</script>

<div class="toasts" role="status" aria-live="polite">
  {#each app.toasts as t (t.id)}
    <div class="toast {t.kind}">
      <Icon name={t.kind === 'error' ? 'warn' : t.kind === 'ok' ? 'check' : 'info'} />
      <span>{t.text}</span>
    </div>
  {/each}
</div>

<style>
  .toasts {
    position: fixed;
    right: 2rem;
    bottom: 4.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    z-index: 50;
    max-width: min(30rem, 90vw);
  }
  .toast {
    display: flex;
    gap: 0.6rem;
    align-items: flex-start;
    padding: 0.8rem 1rem;
    background: var(--surface);
    border: 3px solid var(--accent-2);
    box-shadow: var(--glow-2);
    white-space: pre-line;
    animation: pop 220ms var(--ease);
  }
  .toast.ok {
    border-color: #5dc44f;
  }
  .toast.error {
    border-color: #ff6b6b;
  }
  @keyframes pop {
    from {
      transform: translateY(12px);
      opacity: 0;
    }
  }
</style>
