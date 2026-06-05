import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, type Mock } from 'vitest';
import SchemaStatus from './SchemaStatus';
import { useSchemaStatusQuery } from '@/lib/api-client';

// Mock the api-client hook so the component is tested through the authenticated
// data path it actually uses (the hook attaches the session token).
vi.mock('@/lib/api-client', () => ({
  useSchemaStatusQuery: vi.fn(),
}));

const mockStatus = {
  schema_source: 'schemas/event.v1.json',
  declared_types: 36,
  observed_types: 2,
  total_events: 4,
  incomplete_events: 1,
  undeclared_types: ['custom.unknown.type'],
  types: [
    {
      type: 'atb.tool.call',
      criticality: 'required',
      declared: true,
      observed: 2,
      required_fields: ['session_id', 'tool_name'],
      incomplete: 1,
      missing_fields: ['tool_name'],
    },
    {
      type: 'atb.data.export',
      criticality: 'required',
      declared: true,
      observed: 0,
      required_fields: ['session_id', 'export_target'],
      incomplete: 0,
    },
    {
      type: 'custom.unknown.type',
      criticality: '',
      declared: false,
      observed: 1,
      required_fields: [],
      incomplete: 0,
    },
  ],
};

describe('SchemaStatus', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders loading state', () => {
    (useSchemaStatusQuery as Mock).mockReturnValue({ data: undefined, isLoading: true, error: null });
    render(<SchemaStatus />);
    expect(screen.getByText('Loading schema status…')).toBeInTheDocument();
  });

  it('renders error state', () => {
    (useSchemaStatusQuery as Mock).mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error('boom'),
    });
    render(<SchemaStatus />);
    expect(screen.getByText('Error: boom')).toBeInTheDocument();
  });

  it('surfaces summary, undeclared types, and incomplete rows', () => {
    (useSchemaStatusQuery as Mock).mockReturnValue({ data: mockStatus, isLoading: false, error: null });
    render(<SchemaStatus />);

    // Summary cards.
    expect(screen.getByText('Declared types')).toBeInTheDocument();
    expect(screen.getByText('Incomplete events')).toBeInTheDocument();

    // Undeclared banner names the rogue type.
    expect(screen.getByText('Undeclared types observed:')).toBeInTheDocument();
    expect(screen.getAllByText(/custom\.unknown\.type/).length).toBeGreaterThan(0);

    // Incomplete row shows the missing required field and an incomplete label.
    expect(screen.getByText('1 incomplete')).toBeInTheDocument();
    expect(screen.getByText(/missing: tool_name/)).toBeInTheDocument();

    // Unobserved declared type shows a not-observed status.
    expect(screen.getByText('not observed')).toBeInTheDocument();
  });
});
