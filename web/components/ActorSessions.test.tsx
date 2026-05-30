import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import { useQuery } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import ActorSessions from './ActorSessions';
import AnomalyBadge from './AnomalyBadge'; // Import AnomalyBadge for mocking

// Mock @tanstack/react-query
vi.mock('@tanstack/react-query', () => ({
  useQuery: vi.fn(),
}));

// Mock next/navigation
vi.mock('next/navigation', () => ({
  useRouter: vi.fn(),
}));

// Mock AnomalyBadge to simplify testing its parent
vi.mock('./AnomalyBadge', () => ({
  default: vi.fn(({ flag }) => <span data-testid={`anomaly-badge-${flag}`} title={`Mock ${flag}`} />),
}));


const mockActorSessions = {
  'User One': [
    {
      session_id: 's1',
      actor: { display_name: 'User One', email: 'user1@example.com' },
      started_at: '2026-05-28T10:00:00Z',
      closed_at: '2026-05-28T10:05:00Z',
      exchange_count: 5,
      inferred_profile: 'atb.profile.privileged_tool_action',
      cas_grade: 'High',
      anomaly_flags: [],
      bundle_path: '/path/to/bundle1.atb',
    },
    {
      session_id: 's3',
      actor: { display_name: 'User One', email: 'user1@example.com' },
      started_at: '2026-05-28T12:00:00Z',
      closed_at: null,
      exchange_count: 3,
      inferred_profile: 'atb.profile.rag',
      cas_grade: 'Medium',
      anomaly_flags: ['session_not_closed'],
      bundle_path: '/path/to/bundle3.atb',
    },
  ],
  'api-key:abcd': [
    {
      session_id: 's2',
      actor: { display_name: 'api-key:abcd', email: '' },
      started_at: '2026-05-28T11:00:00Z',
      closed_at: null,
      exchange_count: 2,
      inferred_profile: 'atb.profile.data_export',
      cas_grade: 'Low',
      anomaly_flags: ['unresolved_identity', 'session_not_closed'],
      bundle_path: '/path/to/bundle2.atb',
    },
  ],
};

describe('ActorSessions', () => {
  const mockPush = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    (useRouter as Mock).mockReturnValue({
      push: mockPush,
    });
    Object.defineProperty(window, 'location', {
      value: {
        hash: '#session=test-token',
      },
      writable: true,
    });
  });

  it('renders loading state', () => {
    (useQuery as Mock).mockReturnValue({
      data: undefined,
      isLoading: true,
      error: null,
    });
    render(<ActorSessions />);
    expect(screen.getByText('Loading actor sessions...')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useQuery as Mock).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('Failed to fetch'),
    });
    render(<ActorSessions />);
    expect(screen.getByText('Error: Failed to fetch')).toBeInTheDocument();
  });

  it('renders no actor sessions found state', () => {
    (useQuery as Mock).mockReturnValue({
      data: {},
      isLoading: false,
      error: null,
    });
    render(<ActorSessions />);
    expect(screen.getByText('No actor sessions found.')).toBeInTheDocument();
  });

  it('renders actor groups and expands/collapses', async () => {
    (useQuery as Mock).mockReturnValue({
      data: mockActorSessions,
      isLoading: false,
      error: null,
    });
    render(<ActorSessions />);

    // Initially collapsed
    expect(screen.getByText('User One')).toBeInTheDocument();
    expect(screen.getByText('api-key:abcd')).toBeInTheDocument();
    expect(screen.queryByText('s1')).not.toBeInTheDocument();

    // Expand User One
    fireEvent.click(screen.getByText('User One'));
    await waitFor(() => {
      expect(screen.getByText('s1')).toBeInTheDocument();
      expect(screen.getByText('s3')).toBeInTheDocument();
    });

    // Collapse User One
    fireEvent.click(screen.getByText('User One'));
    await waitFor(() => {
      expect(screen.queryByText('s1')).not.toBeInTheDocument();
      expect(screen.queryByText('s3')).not.toBeInTheDocument();
    });
  });

  it('displays unresolved_identity badge for actor group if any session has it', async () => {
    (useQuery as Mock).mockReturnValue({
      data: mockActorSessions,
      isLoading: false,
      error: null,
    });
    render(<ActorSessions />);

    // Check for badge on 'api-key:abcd' group
    const apiKeyActorHeader = screen.getByText('api-key:abcd').closest('div');
    expect(apiKeyActorHeader).toBeInTheDocument();
    expect(apiKeyActorHeader?.querySelector('[data-testid="anomaly-badge-unresolved_identity"]')).toBeInTheDocument();

    // Check no badge on 'User One' group
    const userOneActorHeader = screen.getByText('User One').closest('div');
    expect(userOneActorHeader).toBeInTheDocument();
    expect(userOneActorHeader?.querySelector('[data-testid="anomaly-badge-unresolved_identity"]')).not.toBeInTheDocument();
  });

  it('navigates to bundle viewer on session row click', async () => {
    (useQuery as Mock).mockReturnValue({
      data: mockActorSessions,
      isLoading: false,
      error: null,
    });
    render(<ActorSessions />);

    fireEvent.click(screen.getByText('User One')); // Expand
    await waitFor(() => {
      fireEvent.click(screen.getByText('s1'));
    });

    expect(mockPush).toHaveBeenCalledWith(
      '/view?bundlePath=%2Fpath%2Fto%2Fbundle1.atb#session=test-token'
    );
  });
});
