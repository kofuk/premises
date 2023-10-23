import useSWR from 'swr';

import {SessionData, WorldBackup} from './entities';

const getSessionData = async (): SessionData => {
  const resp = await fetch('/api/session-data').then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export const useSessionData = () => {
  const {data, mutate, isLoading} = useSWR('/api/session-data', getSessionData);
  return {
    session: data,
    isLoading,
    mutate
  };
};

const getBackups = async (): WorldBackup[] => {
  const resp = await fetch('/api/backups').then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export const useBackups = () => {
  const {data, mutate, isLoading} = useSWR('/api/backups', getBackups);
  return {
    backups: data,
    mutate,
    isLoading
  };
};
