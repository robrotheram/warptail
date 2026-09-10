import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/settings')({
  validateSearch: (search: Record<string, unknown>): { tab: 'table' | 'logs' | 'status' } => ({
    tab: search.tab === 'logs' || search.tab === 'status' ? search.tab : 'table',
  }),
});
