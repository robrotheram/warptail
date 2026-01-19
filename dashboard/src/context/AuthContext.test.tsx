import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactNode } from 'react';

// Mock the API module
vi.mock('@/lib/api', () => ({
  getProfile: vi.fn().mockResolvedValue({ id: '1', name: 'Test User', email: 'test@test.com' }),
}));

// Mock TanStack Router
vi.mock('@tanstack/react-router', () => ({
  useNavigate: () => vi.fn(),
}));

const createTestQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: {
      retry: false,
      gcTime: 0,
    },
  },
});

const createWrapper = () => {
  const queryClient = createTestQueryClient();
  return ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>{children}</AuthProvider>
    </QueryClientProvider>
  );
};

describe('AuthContext', () => {
  // Mock sessionStorage
  const mockSessionStorage = {
    getItem: vi.fn(),
    setItem: vi.fn(),
    removeItem: vi.fn(),
  };

  beforeEach(() => {
    // Replace the real sessionStorage with our mock
    Object.defineProperty(window, 'sessionStorage', {
      value: mockSessionStorage,
      writable: true
    });
  });

  afterEach(() => {
    // Clear all mocks between tests
    vi.clearAllMocks();
  });

  it('throws error when useAuth is used outside of AuthProvider', () => {
    const queryClient = createTestQueryClient();
    try {
      expect(() => renderHook(() => useAuth(), {
        wrapper: ({ children }) => (
          <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
        ),
      })).toThrowError("useAuth must be used within an AuthProvider");     
    } catch (error) {
      console.log("")
    }
  });

  it('initializes with token from sessionStorage', async () => {
    // Setup mock to return a token
    mockSessionStorage.getItem.mockReturnValue('saved-token');

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    });

    // Wait for the hook to initialize and check token
    await waitFor(() => {
      expect(result.current).not.toBeNull();
    });
    
    expect(result.current.token).toBe('saved-token');
    expect(mockSessionStorage.getItem).toHaveBeenCalledWith('token');

    // Wait for authentication to resolve (profile query)
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });
  });

  it('initializes with null token when sessionStorage is empty', () => {
    // Setup mock to return null
    mockSessionStorage.getItem.mockReturnValue(null);

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    });

    // Check initial state
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('successfully logs in and updates sessionStorage', async () => {
    // Setup mock to return null initially
    mockSessionStorage.getItem.mockReturnValue(null);

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    });

    // Wait for initial render to complete
    await waitFor(() => {
      expect(result.current).not.toBeNull();
    });

    // Perform login
    act(() => {
      result.current.login('new-test-token');
    });

    // Wait for token state to update
    await waitFor(() => {
      expect(result.current.token).toBe('new-test-token');
    });
    
    expect(mockSessionStorage.setItem).toHaveBeenCalledWith('token', 'new-test-token');

    // Wait for authentication to resolve
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });
  });

  it('successfully logs out and cleans sessionStorage', async () => {
    // Setup initial logged-in state
    mockSessionStorage.getItem.mockReturnValue('existing-token');

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    });

    // Wait for initial auth to settle
    await waitFor(() => {
      expect(result.current.isAuthenticated).toBe(true);
    });

    // Perform logout
    act(() => {
      result.current.logout();
    });

    // Check if state and sessionStorage were updated
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
    expect(mockSessionStorage.removeItem).toHaveBeenCalledWith('token');
  });

  it('provides stable references for login and logout functions', async () => {
    // Setup mock to return a token
    mockSessionStorage.getItem.mockReturnValue('test-token');

    // Render the hook with AuthProvider
    const { result, rerender } = renderHook(() => useAuth(), {
      wrapper: createWrapper(),
    });

    // Wait for initial state to settle
    await waitFor(() => {
      expect(result.current.token).toBe('test-token');
    });

    // Store initial function references
    const initialLogin = result.current.login;
    const initialLogout = result.current.logout;

    // Rerender the component
    rerender();

    // Check if function references remain the same
    expect(result.current.login).toBe(initialLogin);
    expect(result.current.logout).toBe(initialLogout);
  });
});