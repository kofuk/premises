import {t} from 'i18next';
import useSWR, {KeyedMutator} from 'swr';
import useSWRImmutable from 'swr/immutable';

import {
  CredentialAssertionResponse,
  CredentialNameAndCreationResponse,
  MCVersion,
  Passkey,
  PasswordCredential,
  SessionData,
  SessionState,
  UpdatePassword,
  WorldBackup
} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

export class APIError extends Error {}

const api = async <T, U>(endpoint: string, method: string = 'get', body?: T) => {
  const options = {method} as any;
  if (body) {
    options.body = JSON.stringify(body);
  }

  const resp = await fetch(`${domain}${endpoint}`, options).then((resp) => resp.json());

  if (!resp.success) {
    throw new APIError(t(`error.code_${resp.errorCode}`));
  }

  return resp.data as U;
};

export default api;

const declareApi =
  <T, U>(endpoint: string, method: string = 'get') =>
  (body?: T) =>
    api<T, U>(endpoint, method, body);

export const login = declareApi<PasswordCredential, SessionState>('/login', 'post');
export const getSessionData = declareApi<null, SessionData>('/api/session-data');
export const getBackups = declareApi<null, WorldBackup[]>('/api/backups');
export const getMCVersions = declareApi<null, MCVersion[]>('/api/mcversions');
export const getPasskeys = declareApi<null, Passkey[]>('/api/hardwarekey');
export const changePassword = declareApi<UpdatePassword, null>('/api/users/change-password', 'post');
export const getPasskeysRegistrationOptions = declareApi<null, CredentialCreationOptions>('/api/hardwarekey/begin', 'post');
export const registerPasskeys = declareApi<CredentialNameAndCreationResponse, null>('/api/hardwarekey/finish', 'post');
export const getPasskeysLoginOptions = declareApi<null, CredentialRequestOptions>('/login/hardwarekey/begin', 'post');
export const loginPasskeys = declareApi<CredentialAssertionResponse, null>('/login/hardwarekey/finish', 'post');
export const addUser = declareApi<PasswordCredential, null>('/api/users/add', 'post');

export type ImmutableUseResponse<T> = {
  data: T | undefined;
  error: Error;
  isLoading: boolean;
};

export type MutableUseResponse<T> = ImmutableUseResponse<T> & {
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
    mutate(
      async (): Promise<undefined> => {
        api<null, null>(`/api/hardwarekey/${id}`, 'delete');
      },
      {
        optimisticData: data && data.filter((key) => key.id != id),
        populateCache: false
      }
    );
  };

  return {
    data,
    error,
    isLoading,
    mutate,
    deleteKey
  };
};
