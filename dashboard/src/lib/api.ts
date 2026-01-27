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

export const token = {
    set: (newToken: string) => sessionStorage.setItem('token', newToken),
    get: () => sessionStorage.getItem('token'),
    remove: () => sessionStorage.removeItem('token')
}

let BASE_URL = ""
if (isDev) {
    BASE_URL = "http://localhost:8001"
}
export const API_URL = `${BASE_URL}/api`
export const AUTH_URL = `${BASE_URL}/auth`

const getAuth = (tkn?: string | null) => {
    return {
        Authorization: token.get() ?? tkn
    } as Record<string, string>
}




export const login = async (login: Login): Promise<LoginToken> => {
    const response = await axios.post(`${AUTH_URL}/login`, login);
    if (response.status !== 200) throw new Error("Unauthorized");
    return response.data;
}

export const getConfig = async (): Promise<Config> => {
    const response = await axios.get(`${BASE_URL}/config`, {
        headers: getAuth(),
    });
    return response.data;
}

// GET SERVICES
export const getServices = async (): Promise<Service[]> => {
    const response = await axios.get(`${API_URL}/services`, {
        headers: getAuth(),
    });
    return response.data;
}

// CREATE SERVICE
export const createService = async (route: CreateService): Promise<Service> => {
    const response = await axios.post(`${API_URL}/services`, route, {
        headers: getAuth(),
    });
    return response.data;
}

// GET A SPECIFIC SERVICE
export const getService = async (name: string): Promise<Service> => {
    const response = await axios.get(`${API_URL}/services/${name}`, {
        headers: getAuth(),
    });
    return response.data;
}

// UPDATE SERVICE
export const updateService = async (svc: Service): Promise<Service> => {
    const response = await axios.put(`${API_URL}/services/${svc.id}`, svc, {
        headers: getAuth(),
    });
    return response.data;
}

// DELETE SERVICE
export const deleteService = async (svc: Service): Promise<void> => {
    await axios.delete(`${API_URL}/services/${svc.id}`, {
        headers: getAuth(),
    });
}

// START SERVICE
export const startService = async (name: string): Promise<Service> => {
    const response = await axios.post(`${API_URL}/services/${name}/start`, {}, {
        headers: getAuth(),
    });
    return response.data;
}

// STOP SERVICE
export const stopService = async (name: string): Promise<Service> => {
    const response = await axios.post(`${API_URL}/services/${name}/stop`, {}, {
        headers: getAuth(),
    });
    return response.data;
}

// GET TAILSALE CONFIGURATION
export const getTSConfig = async (): Promise<Tailsale> => {
    const response = await axios.get(`${API_URL}/settings/tailscale`, {
        headers: getAuth(),
    });
    return response.data;
}

export const getLogs = async (type:string): Promise<string[]> => {
    const response = await axios.get(`${API_URL}/settings/logs?type=${type}`, {
        headers: getAuth(),
    });
    return response.data;
}


export const getTSSTATUS = async (): Promise<TS_STATUS> => {
    const response = await axios.get(`${API_URL}/settings/tailscale/status`, {
        headers: getAuth(),
    });
    return response.data;
}

// UPDATE TAILSALE CONFIGURATION
export const updateTSConfig = async (config: Tailsale): Promise<Tailsale> => {
    const response = await axios.post(`${API_URL}/settings/tailscale`, config, {
        headers: getAuth(),
    });
    return response.data;
}



export const getUsers = async (): Promise<User[]> => {
    const response = await axios.get(`${API_URL}/user`, {
        headers: getAuth(),
    });
    return response.data;
}

export const createUser = async (user: User): Promise<User> => {
    await axios.put(`${API_URL}/user`, user, {
        headers: getAuth(),
    });
    return user;
}

export const updateUser = async (user: User): Promise<User> => {
    console.log("HELLO")
    await axios.post(`${API_URL}/user/${user.id}`, user, {
        headers: getAuth(),
    });
    return user;
}

export const deleteUser = async (user: User): Promise<User> => {
    await axios.delete(`${API_URL}/user/${user.id}`, {
        headers: getAuth(),
    });
    return user;
}


export const getProfile = async (token?: string | null): Promise<User> => {
    const response = await axios.get(`${AUTH_URL}/profile`, {
        headers: getAuth(token),
    });
    return response.data;
}


export const getTailScaleNodes = async (): Promise<TailsaleNode[]> => {
    const response = await axios.get(`${API_URL}/tailsale/nodes`, {
        headers: getAuth(),
    });
    return response.data;
}
