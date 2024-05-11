import {t} from 'i18next';
import useSWR, {KeyedMutator} from 'swr';
import useSWRImmutable from 'swr/immutable';

import {
  ConfigAndValidity,
  MCVersion,
  PasswordCredential,
  PendingConfig,
  SessionData,
  SessionState,
  SnapshotConfiguration,
  SystemInfo,
  UpdatePassword,
  WorldBackup,
  WorldInfo
} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

export class APIError extends Error {}

const api = async <T, U>(endpoint: string, method: string = 'get', body?: T) => {
  const options = {method} as any;
  if (body) {
    options.body = JSON.stringify(body);
    options.headers = {
      'Content-Type': 'application/json'
    };
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
export const changePassword = declareApi<UpdatePassword, null>('/api/users/change-password', 'post');
export const addUser = declareApi<PasswordCredential, null>('/api/users/add', 'post');
export const getSystemInfo = declareApi<null, SystemInfo>('/api/systeminfo');
export const getWorldInfo = declareApi<null, WorldInfo>('/api/worldinfo');
export const takeQuickSnapshot = declareApi<SnapshotConfiguration, null>('/api/quickundo/snapshot', 'post');
export const undoQuickSnapshot = declareApi<SnapshotConfiguration, null>('/api/quickundo/undo', 'post');
export const getConfig = declareApi<null, ConfigAndValidity>('/api/config');
export const updateConfig = declareApi<PendingConfig, ConfigAndValidity>('/api/config', 'put');
export const launch = declareApi<null, null>('/api/launch', 'post');
export const reconfigure = declareApi<null, null>('/api/reconfigure', 'post');

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
