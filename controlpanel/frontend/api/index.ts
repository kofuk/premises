export const getSessionData = async (): SessionData => {
  return await fetch('/api/session-data').then((resp) => resp.json());
};
