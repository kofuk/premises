import {t} from 'i18next';
import useSWR, {KeyedMutator} from 'swr';
import useSWRImmutable from 'swr/immutable';

import type {
  ConfigAndValidity,
  CreateWorldDownloadLinkReq,
  CreateWorldUploadLinkReq,
  DelegatedURL,
  DeleteWorldInput,
  MCVersion,
  PasswordCredential,
  PendingConfig,
  SessionData,
  SessionState,
  SnapshotConfiguration,
  SystemInfo,
  UpdatePassword,
  World,
  WorldInfo
} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

export class APIError extends Error {}

const api = async <T, U>(endpoint: string, method: string = 'get', accessToken: string | null, body?: T) => {
  const options = {method} as any;
  if (body) {
    options.body = JSON.stringify(body);
    options.headers = {
      'Content-Type': 'application/json'
    };
  }
  if (accessToken) {
    options.headers = {
      ...options.headers,
      Authorization: `Bearer ${accessToken}`
    };
  }

  const resp = await fetch(`${domain}${endpoint}`, options);
  if (!resp.ok) {
    throw new APIError(`${resp.status} ${resp.statusText}`);
  }

  const json = await resp.json();
  if (!json.success) {
    throw new APIError(t(`error.code_${json.errorCode}`));
  }

  return json.data as U;
};

export default api;

const declareApi =
  <T, U>(endpoint: string, method: string = 'get') =>
  (accessToken: string | null, body?: T) =>
    api<T, U>(endpoint, method, accessToken, body);

export const login = declareApi<PasswordCredential, SessionState>('/api/internal/login', 'post');
export const getSessionData = declareApi<null, SessionData>('/api/internal/session-data');
export const listWorlds = declareApi<null, World[]>('/api/v1/worlds');
export const getMCVersions = declareApi<null, MCVersion[]>('/api/v1/mcversions');
export const changePassword = declareApi<UpdatePassword, null>('/api/v1/users/change-password', 'post');
export const addUser = declareApi<PasswordCredential, null>('/api/v1/users/add', 'post');
export const getSystemInfo = declareApi<null, SystemInfo>('/api/v1/systeminfo');
export const getWorldInfo = declareApi<null, WorldInfo>('/api/v1/worldinfo');
export const takeQuickSnapshot = declareApi<SnapshotConfiguration, null>('/api/v1/quickundo/snapshot', 'post');
export const undoQuickSnapshot = declareApi<SnapshotConfiguration, null>('/api/v1/quickundo/undo', 'post');
export const getConfig = declareApi<null, ConfigAndValidity>('/api/v1/config');
export const updateConfig = declareApi<PendingConfig, ConfigAndValidity>('/api/v1/config', 'put');
export const launch = declareApi<null, null>('/api/v1/launch', 'post');
export const reconfigure = declareApi<null, null>('/api/v1/reconfigure', 'post');
export const stop = declareApi<null, null>('/api/v1/stop', 'post');
export const createWorldDownloadLink = declareApi<CreateWorldDownloadLinkReq, DelegatedURL>('/api/v1/world-link/download', 'post');
export const createWorldUploadLink = declareApi<CreateWorldUploadLinkReq, DelegatedURL>('/api/v1/world-link/upload', 'post');
export const deleteWorld = declareApi<DeleteWorldInput, null>('/api/v1/worlds', 'delete');

export type ImmutableUseResponse<T> = {
  data: T | undefined;
  error: Error;
  isLoading: boolean;
};

export type MutableUseResponse<T> = ImmutableUseResponse<T> & {
  mutate: KeyedMutator<T>;
};

export const useSessionData = (accessToken: string | null): MutableUseResponse<SessionData> => {
  const {data, error, mutate, isLoading} = useSWR('/session-data', () => getSessionData(accessToken));
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

export const useWorlds = (accessToken: string | null): MutableUseResponse<World[]> => {
  const {data, error, isLoading, mutate} = useSWR('/api/worlds', () => listWorlds(accessToken));
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

export const useMCVersions = (accessToken: string | null): ImmutableUseResponse<MCVersion[]> => {
  const {data, error, isLoading} = useSWRImmutable('/api/mcversions', () => getMCVersions(accessToken));
  return {
    data,
    error,
    isLoading
  };
};
