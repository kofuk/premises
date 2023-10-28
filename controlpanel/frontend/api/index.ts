import useSWR, {KeyedMutator} from 'swr';
import useSWRImmutable from 'swr/immutable';

import {MCVersion, Passkey, SessionData, WorldBackup} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

export type ImmutableUseResponse<T> = {
  data: T | undefined;
  error: Error;
  isLoading: boolean;
};

export type MutableUseResponse<T> = ImmutableUseResponse & {
  mutate: KeyedMutator<T>;
};

const getSessionData = async (): Promise<SessionData> => {
  const resp = await fetch(`${domain}/api/session-data`).then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export const useSessionData = (): MutableUseResponse<SessionData> => {
  const {data, error, mutate, isLoading} = useSWR('/api/session-data', getSessionData);
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

const getBackups = async (): Promise<WorldBackup[]> => {
  const resp = await fetch(`${domain}/api/backups`).then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export const useBackups = (): MutableUseResponse<WorldBackup[]> => {
  const {data, error, isLoading, mutate} = useSWR('/api/backups', getBackups);
  return {
    data,
    error,
    isLoading,
    mutate
  };
};

const getMCVersions = async (): Promise<MCVersion[]> => {
  const resp = await fetch(`${domain}/api/mcversions`).then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export type UseMCVersionsResponse = {
  mcVersions: MCVersion[] | undefined;
  error: Error;
  isLoading: boolean;
};

export const useMCVersions = (): ImmutableUseResponse<MCVersion[]> => {
  const {data, error, isLoading} = useSWRImmutable('/api/mcversions', getMCVersions);
  return {
    data,
    error,
    isLoading
  };
};

const getPasskeys = async (): Promise<Passkey[]> => {
  const resp = await fetch(`${domain}/api/hardwarekey`).then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export type UsePasskeysResponse = MutableUseResponse<Passkey[]> & {
  deleteKey: (id: string) => void;
};

export const usePasskeys = (): UsePasskeysResponse => {
  const {data, error, isLoading, mutate} = useSWR('/api/hardwarekey', getPasskeys);

  const deleteKey = (id: string) => {
    mutate(
      async () => {
        const resp = await fetch(`${domain}/api/hardwarekey/${id}`, {method: 'delete'});
        if (resp.status !== 204) {
          throw new Error('Error deleting key');
        }
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
