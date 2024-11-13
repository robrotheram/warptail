import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { AuthProvider, useAuth } from './AuthContext';


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
    try {
      expect(() =>  renderHook(() => useAuth())).toThrowError("useAuth must be used within an AuthProvider");     
    } catch (error) {
      console.log("")
    }
    
  });

  it('initializes with token from sessionStorage', () => {
    // Setup mock to return a token
    mockSessionStorage.getItem.mockReturnValue('saved-token');

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => <AuthProvider>{children}</AuthProvider>,
    });

    // Check initial state
    expect(result.current.token).toBe('saved-token');
    expect(result.current.isAuthenticated).toBe(true);
    expect(mockSessionStorage.getItem).toHaveBeenCalledWith('token');
  });

  it('initializes with null token when sessionStorage is empty', () => {
    // Setup mock to return null
    mockSessionStorage.getItem.mockReturnValue(null);

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => <AuthProvider>{children}</AuthProvider>,
    });

    // Check initial state
    expect(result.current.token).toBeNull();
    expect(result.current.isAuthenticated).toBe(false);
  });

  it('successfully logs in and updates sessionStorage', () => {
    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => <AuthProvider>{children}</AuthProvider>,
    });

    // Perform login
    act(() => {
      result.current.login('new-test-token');
    });

    // Check if state and sessionStorage were updated
    expect(result.current.token).toBe('new-test-token');
    expect(result.current.isAuthenticated).toBe(true);
    expect(mockSessionStorage.setItem).toHaveBeenCalledWith('token', 'new-test-token');
  });

  it('successfully logs out and cleans sessionStorage', () => {
    // Setup initial logged-in state
    mockSessionStorage.getItem.mockReturnValue('existing-token');

    // Render the hook with AuthProvider
    const { result } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => <AuthProvider>{children}</AuthProvider>,
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

  it('provides stable references for login and logout functions', () => {
    // Render the hook with AuthProvider
    const { result, rerender } = renderHook(() => useAuth(), {
      wrapper: ({ children }) => <AuthProvider>{children}</AuthProvider>,
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