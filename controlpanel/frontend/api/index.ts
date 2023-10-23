import useSWR, {KeyedMutator} from 'swr';

import {SessionData, WorldBackup} from './entities';

const domain = process.env.NODE_ENV === 'test' ? 'http://localhost' : '';

const getSessionData = async (): Promise<SessionData> => {
  const resp = await fetch(`${domain}/api/session-data`).then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export type UseSessionDataResponse = {
  session: SessionData | undefined;
  error: Error;
  isLoading: boolean;
  mutate: KeyedMutator<SessionData>;
};

export const useSessionData = (): UseSessionDataResponse => {
  const {data, error, mutate, isLoading} = useSWR('/api/session-data', getSessionData);
  return {
    session: data,
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

export type UseBackupsResponse = {
  backups: WorldBackup[] | undefined;
  error: Error;
  isLoading: boolean;
  mutate: KeyedMutator<WorldBackup[]>;
};

export const useBackups = (): UseBackupsResponse => {
  const {data, error, isLoading, mutate} = useSWR('/api/backups', getBackups);
  return {
    backups: data,
    error,
    isLoading,
    mutate
  };
};
