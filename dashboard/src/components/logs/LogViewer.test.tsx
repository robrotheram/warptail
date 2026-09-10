import { render, screen, cleanup } from '@testing-library/react';
import { afterEach, expect, it } from 'vitest';
import { LogViewer, logLines, MAX_LOG_LINES } from './LogViewer';

afterEach(cleanup);
it('bounds log history and preserves repeated lines', () => {
  const lines = Array.from({ length: MAX_LOG_LINES + 20 }, (_, index) => `line ${index}`);
  expect(logLines(lines)).toHaveLength(MAX_LOG_LINES);
  expect(logLines(lines)[0]).toBe('line 20');
  expect(logLines(['same', 'same'])).toEqual(['same', 'same']);
});
it('renders hostile HTML as text and only makes HTTP URLs clickable', () => {
  const { container } = render(<LogViewer logs={['<img src=x onerror=alert(1)>', 'javascript:alert(1)', '\u001b[31mError\u001b[0m https://example.com']} />);
  expect(container.querySelector('img')).toBeNull();
  expect(screen.getByText('<img src=x onerror=alert(1)>')).toBeInTheDocument();
  expect(screen.getAllByRole('link')).toHaveLength(1);
  expect(screen.getByRole('link')).toHaveAttribute('rel', 'noopener noreferrer');
});
