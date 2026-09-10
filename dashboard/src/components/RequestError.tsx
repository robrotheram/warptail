import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Button } from '@/components/ui/button';
import { errorMessage } from '@/lib/api';

export function RequestError({ error, title = 'Unable to load data', retry }: {
  error?: unknown; title?: string; retry?: () => void;
}) {
  return <Alert variant="destructive" className="my-4">
    <AlertCircle className="h-4 w-4" />
    <AlertTitle>{title}</AlertTitle>
    <AlertDescription className="space-y-3">
      <p>{errorMessage(error)}</p>
      {retry && <Button variant="outline" size="sm" onClick={retry}>Try again</Button>}
    </AlertDescription>
  </Alert>;
}
