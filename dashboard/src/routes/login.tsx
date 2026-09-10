import { createFileRoute } from '@tanstack/react-router';
import { safeRedirect } from '@/lib/redirect';

export const Route = createFileRoute('/login')({
  validateSearch: (search: Record<string, unknown>): { next?: string } => ({ next: safeRedirect(search.next) ?? undefined }),
});
