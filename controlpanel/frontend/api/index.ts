import useSWR, {KeyedMutator} from 'swr';
import useSWRImmutable from 'swr/immutable';

import {MCVersion, Passkey, PasswordCredential, SessionData, SessionState, WorldBackup} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

const api =
  <T, U>(endpoint: string, method: string = 'get') =>
  async (body: T | undefined): Promise<U> => {
    const options = {method};
    if (body) {
      options.body = JSON.stringify(body);
    }

    const resp = await fetch(`${domain}${endpoint}`, options).then((resp) => resp.json());

    if (!resp.success) {
      throw new Error(resp.reason || resp.errorCode);
    }

    return resp.data as U;
  };

export default api;

export const login = api<PasswordCredential, SessionState>('/login', 'post');
export const getSessionData = api<null, SessionData>('/api/session-data');
export const getBackups = api<null, WorldBackup[]>('/api/backups');
export const getMCVersions = api<null, MCVersion[]>('/api/mcversions');
export const getPasskeys = api<null, Passkey[]>('/api/hardwarekey');

export type ImmutableUseResponse<T> = {
  data: T | undefined;
  error: Error;
  isLoading: boolean;
};

export type MutableUseResponse<T> = ImmutableUseResponse & {
  mutate: KeyedMutator<T>;
};

export const useSessionData = (): MutableUseResponse<SessionData> => {
  const {data, error, mutate, isLoading} = useSWR('/api/session-data', () => getSessionData());
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

export const useBackups = (): MutableUseResponse<WorldBackup[]> => {
  const {data, error, isLoading, mutate} = useSWR('/api/backups', () => getBackups());
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

export const useMCVersions = (): ImmutableUseResponse<MCVersion[]> => {
  const {data, error, isLoading} = useSWRImmutable('/api/mcversions', () => getMCVersions());
  return {
    data,
    error,
    isLoading
  };
};

export type UsePasskeysResponse = MutableUseResponse<Passkey[]> & {
  deleteKey: (id: string) => void;
};

export const usePasskeys = (): UsePasskeysResponse => {
  const {data, error, isLoading, mutate} = useSWR('/api/hardwarekey', () => getPasskeys());

  const deleteKey = (id: string) => {
    mutate(api<null, null>(`/api/hardwarekey/${id}`, 'delete'), {
      optimisticData: data && data.filter((key) => key.id != id),
      populateCache: false
    });
  };

  return {
    data,
    error,
    isLoading,
    mutate,
    deleteKey
  };
};
