export type SessionData = {
  loggedIn: boolean;
  userName: string;
};

export type GenerationInfo = {
  gen: string;
  id: string;
  timestamp: number;
};

export type WorldBackup = {
  worldName: string;
  generations: GenerationInfo[];
};

export type MCVersion = {
  name: string;
  isStable: boolean;
  channel: string;
  releaseDate: string;
};

export type PasswordCredential = {
  userName: string;
  password: string;
};

export type SessionState = {
  needsChangePassword: boolean;
};

export type UpdatePassword = {
  password: string;
  newPassword: string;
};

export type SystemInfo = {
  premisesVersion: string;
  hostOs: string;
  ipAddr: string | null;
};

export type WorldInfo = {
  version: string;
  worldName: string;
  seed: string;
};

export type StatusExtraData = {
  progress: number;
  textData: string;
};
