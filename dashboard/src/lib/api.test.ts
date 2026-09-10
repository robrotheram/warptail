import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AxiosError, AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import { getServices, http, onSessionExpired, token, updateService, Service } from './api';

const originalAdapter = http.defaults.adapter;
const response = (config: InternalAxiosRequestConfig, status = 200): AxiosResponse => ({ config, status, statusText: '', headers: {}, data: [] });
beforeEach(() => token.set('first-account'));
afterEach(() => { token.remove(); http.defaults.adapter = originalAdapter; });

describe('request session isolation', () => {
  it('expires the current session on 401, but keeps 403 as a permission error', async () => {
    const expired = vi.fn();
    const unsubscribe = onSessionExpired(expired);
    http.defaults.adapter = async config => { throw new AxiosError('Forbidden', undefined, config, undefined, response(config, 403)); };
    await expect(getServices()).rejects.toBeDefined();
    expect(expired).not.toHaveBeenCalled();
    http.defaults.adapter = async config => { throw new AxiosError('Unauthorized', undefined, config, undefined, response(config, 401)); };
    await expect(getServices()).rejects.toBeDefined();
    expect(expired).toHaveBeenCalledOnce();
    unsubscribe();
  });

  it('ignores a previous account’s delayed 401', async () => {
    const expired = vi.fn();
    const unsubscribe = onSessionExpired(expired);
    let reject!: (reason: unknown) => void;
    let requestConfig!: InternalAxiosRequestConfig;
    http.defaults.adapter = config => { requestConfig = config; return new Promise((_resolve, fail) => { reject = fail; }); };
    const request = getServices();
    await vi.waitFor(() => expect(reject).toBeDefined());
    token.set('second-account');
    reject(new AxiosError('Unauthorized', undefined, requestConfig, undefined, response(requestConfig, 401)));
    await expect(request).rejects.toBeDefined();
    expect(expired).not.toHaveBeenCalled();
    unsubscribe();
  });

  it('rejects a stale successful mutation before its callbacks can refill caches', async () => {
    let finish!: () => void;
    http.defaults.adapter = config => new Promise(resolve => { finish = () => resolve(response(config)); });
    const request = updateService({ id: 'example' } as Service);
    await vi.waitFor(() => expect(finish).toBeDefined());
    token.remove();
    token.set('first-account'); // Even reusing an identical token starts a different session.
    finish();
    await expect(request).rejects.toMatchObject({ code: 'ERR_CANCELED' });
  });
});
