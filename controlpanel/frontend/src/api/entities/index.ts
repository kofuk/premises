export type SessionData = {
  loggedIn: boolean;
  accessToken: string;
};

export type WorldGeneration = {
  gen: string;
  id: string;
  timestamp: number;
};

export type World = {
  worldName: string;
  generations: WorldGeneration[];
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

export type SnapshotConfiguration = {
  slot: number;
};

export type PendingConfig = {
  machineType?: string;
  serverVersion?: string;
  guessServerVersion?: boolean;
  worldSource?: string;
  worldName?: string;
  backupGen?: string;
  levelType?: string;
  seed?: string;
  motd?: string;
  serverPropOverride?: Record<string, string>;
  inactiveTimeout?: number;
};

export type ConfigAndValidity = {
  isValid: boolean;
  config: PendingConfig;
};

export type CreateWorldDownloadLinkReq = {
  id: string;
};

export type CreateWorldUploadLinkReq = {
  worldName: string;
  mimeType: string;
};

export type DelegatedURL = {
  url: string;
};
