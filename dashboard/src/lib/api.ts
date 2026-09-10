import axios from 'axios';
export const isDev = !!import.meta.env.DEV;

export enum RouterStatus {
    STARTING = "Starting",
    RUNNING = "Running",
    STOPPED = "Stopped",
}

export enum RouterType {
    HTTPS = "https",
    HTTP = "http",
    TCP = "tcp",
    UDP = "udp",
}

export enum Role {
    ADMIN = "admin",
    USER = "user",
}

export enum TS_STATE {
    NOSTATE = "NoState",
    NEEDS_LOGIN = "NeedsLogin",
    NEEDS_MACHINE_AUTH = "NeedsMachineAuth",
    STOPPED = "Stopped",
    STARTING = "Starting",
    RUNNING = "Running"
}

export interface TS_STATUS {
    messages: string[]
    version: string
    state: TS_STATE
    nodes: TailsaleNode[]
    hostname: string
    key_expiry: Date | null
    auth_url?: string
}

export interface Config {
    read_only: boolean
    auth_type: string
    auth_name: string
    site_name?: string
    site_logo?: string
}

export interface TailsaleNode {
    id: string
    name: string
    hostname : string
    ip: string
    online: boolean
    os: string
    key_expiry: Date
    last_seen: string
}

export interface CreateService {
    name: string
    routes: Route[]
}


export interface Service {
    id: string
    name: string
    enabled: boolean
    routes: Route[]
    latency: number
}

export interface ProxyRule {
    path: string
    target_host?: string
    target_port?: number
    rewrite?: string
    strip_path?: boolean
}

export interface ProxyHeaders {
    add?: Record<string, string>
    remove?: string[]
    set?: Record<string, string>
}

export interface ProxySettings {
    timeout?: number
    retry_attempts?: number
    buffer_requests?: boolean
    preserve_host?: boolean
    follow_redirects?: boolean
    custom_headers?: ProxyHeaders
    rules?: ProxyRule[]
}

export interface Route {
    key?: number
    private: boolean
    bot_protect: boolean
    type: string
    domain?: string
    port?: number
    machine: Machine
    status?: RouterStatus
    latency?: number
    proxy_settings?: ProxySettings
    stats?: TimeSeries
}

export interface Machine {
    node?: string
    address: string
    port: number
}

export interface Tailsale {
    AuthKey: string
    Hostname: string
}

export interface ProxyStats {
    sent: number;
    received: number;
}

export interface TimeSeriesPoint {
    timestamp: Date
    value: ProxyStats
}
export interface TimeSeries {
    points: TimeSeriesPoint[]
    total: ProxyStats
}


export interface Dashboard {
    Enabled: boolean
    Username: string
    Password: string
}

export interface Login {
    username: string
    password: string
}


export interface User {
    id?: string
    name: string
    type: string
    password?: string
    email: string
    role?: Role
    created_at?: Date
    password_reset?: boolean
}

export interface LoginToken {
    authorization_token: string
    role: Role
}

// Storage can be unavailable in privacy mode. Keep the current session usable in memory.
let memoryToken: string | null = null;
let sessionVersion = 0;
export const token = {
    set: (value: string) => {
        sessionVersion++;
        memoryToken = value;
        try { sessionStorage.setItem('token', value); } catch { /* Memory fallback. */ }
    },
    get: (): string | null => {
        try { return sessionStorage.getItem('token') ?? memoryToken; } catch { return memoryToken; }
    },
    remove: () => {
        sessionVersion++;
        memoryToken = null;
        try { sessionStorage.removeItem('token'); } catch { /* Memory fallback. */ }
    },
};

export const API_URL = '/api';
export const AUTH_URL = '/auth';
export const http = axios.create({ timeout: 20_000 });
const requestVersions = new WeakMap<object, number>();
http.interceptors.request.use(config => {
    requestVersions.set(config, sessionVersion);
    return config;
});
const expiredListeners = new Set<() => void>();
export function onSessionExpired(listener: () => void) {
    expiredListeners.add(listener);
    return () => { expiredListeners.delete(listener); };
}

http.interceptors.response.use(response => {
    if (response.config.headers.Authorization && requestVersions.get(response.config) !== sessionVersion) {
        throw new axios.CanceledError('Session changed');
    }
    return response;
}, error => {
    // A late response from a previous account must never sign out the current account.
    const credential = error.config?.headers?.Authorization;
    if (axios.isAxiosError(error) && error.response?.status === 401 &&
        credential && credential === token.get() && error.config && requestVersions.get(error.config) === sessionVersion && error.config?.url !== `${AUTH_URL}/login`) {
        expiredListeners.forEach(listener => listener());
    }
    return Promise.reject(error);
});

export function isUnauthorized(error: unknown): boolean {
    return axios.isAxiosError(error) && error.response?.status === 401;
}

export function errorMessage(error: unknown, fallback = 'Unable to complete the request. Please try again.'): string {
    if (!axios.isAxiosError(error)) return fallback;
    if (error.response?.status === 403) return 'You do not have permission to do this.';
    if (error.response?.status === 401) return 'Your session has expired. Please sign in again.';
    if (!error.response) return 'Unable to reach WarpTail. Check your connection and try again.';
    return fallback;
}

const headers = (credential = token.get()) => credential ? { Authorization: credential } : {};
type QueryRequest = { signal?: AbortSignal };
const read = async <T,>(url: string, signal?: AbortSignal, credential = token.get()): Promise<T> =>
    (await http.get<T>(url, { headers: headers(credential), signal })).data;
const write = async <T,>(method: 'post' | 'put' | 'delete', url: string, data?: unknown): Promise<T> =>
    (await http.request<T>({ method, url, data, headers: headers() })).data;
const serviceURL = (id: string) => `${API_URL}/services/${encodeURIComponent(id)}`;

export const login = async (credentials: Login): Promise<LoginToken> =>
    (await http.post<LoginToken>(`${AUTH_URL}/login`, credentials)).data;
export const logout = () => write<void>('post', `${AUTH_URL}/logout`);
export const getConfig = ({ signal }: QueryRequest = {}) => read<Config>('/config', signal, null);
export const getServices = async ({ signal }: QueryRequest = {}) => (await read<Service[]>(`${API_URL}/services`, signal)) ?? [];
export const createService = (service: CreateService) => write<Service>('post', `${API_URL}/services`, service);
export const getService = (id: string, signal?: AbortSignal) => read<Service>(serviceURL(id), signal);
export const updateService = (service: Service) => write<Service>('put', serviceURL(service.id), service);
export const deleteService = (service: Service) => write<void>('delete', serviceURL(service.id));
export const startService = (id: string) => write<Service>('post', `${serviceURL(id)}/start`, {});
export const stopService = (id: string) => write<Service>('post', `${serviceURL(id)}/stop`, {});
export const getTSConfig = ({ signal }: QueryRequest = {}) => read<Tailsale>(`${API_URL}/settings/tailscale`, signal);
export const getTSSTATUS = ({ signal }: QueryRequest = {}) => read<TS_STATUS>(`${API_URL}/settings/tailscale/status`, signal);
export const updateTSConfig = (config: Tailsale) => write<void>('post', `${API_URL}/settings/tailscale`, config);
export const getLogs = async (type: string, signal?: AbortSignal) =>
    (await read<string[]>(`${API_URL}/settings/logs?type=${encodeURIComponent(type)}`, signal)) ?? [];
export const getUsers = async ({ signal }: QueryRequest = {}) => (await read<User[]>(`${API_URL}/user`, signal)) ?? [];
export const createUser = (user: User) => write<void>('put', `${API_URL}/user`, user);
export const updateUser = (user: User) => write<void>('post', `${API_URL}/user/${encodeURIComponent(user.id!)}`, user);
export const updateProfile = (user: User) => write<User>('post', `${AUTH_URL}/profile`, user);
export const deleteUser = (user: User) => write<void>('delete', `${API_URL}/user/${encodeURIComponent(user.id!)}`);
export const getProfile = (credential: string | null = token.get(), signal?: AbortSignal) => read<User>(`${AUTH_URL}/profile`, signal, credential);
export const getTailScaleNodes = async ({ signal }: QueryRequest = {}) => (await read<TailsaleNode[]>(`${API_URL}/tailsale/nodes`, signal)) ?? [];
