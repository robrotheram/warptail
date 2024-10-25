import { createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/404')({
  component: () => <div>Hello /404!</div>,
})
