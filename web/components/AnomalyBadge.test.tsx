import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import AnomalyBadge from './AnomalyBadge';

describe('AnomalyBadge', () => {
  it('renders tool_without_approval badge with correct tooltip', () => {
    render(<AnomalyBadge flag="tool_without_approval" />);
    const badge = screen.getByTitle(
      'No matching earlier approval exists in the captured evidence; this does not prove no approval occurred elsewhere.',
    );
    expect(badge).toBeInTheDocument();
    expect(badge.querySelector('svg')).toHaveClass('text-amber-500');
  });

  it('renders unresolved_identity badge with correct tooltip', () => {
    render(<AnomalyBadge flag="unresolved_identity" />);
    const badge = screen.getByTitle('Actor identity is unresolved (e.g., API key).');
    expect(badge).toBeInTheDocument();
    expect(badge.querySelector('svg')).toHaveClass('text-gray-500');
  });

  it('renders session_not_closed badge with correct tooltip', () => {
    render(<AnomalyBadge flag="session_not_closed" />);
    const badge = screen.getByTitle('Session has not been explicitly closed.');
    expect(badge).toBeInTheDocument();
    expect(badge.querySelector('svg')).toHaveClass('text-red-500');
  });

  it('does not render for unknown flag', () => {
    const { container } = render(<AnomalyBadge flag="unknown_flag" />);
    expect(container).toBeEmptyDOMElement();
  });
});
