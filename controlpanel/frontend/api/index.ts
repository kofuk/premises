export const getSessionData = async (): SessionData => {
  const resp = await fetch('/api/session-data').then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};

export const getBackups = async (): WorldBackup[] => {
  const resp = await fetch('/api/backups').then((resp) => resp.json());
  if (!resp.success) {
    throw new Error(resp.reason);
  }

  return resp.data;
};
