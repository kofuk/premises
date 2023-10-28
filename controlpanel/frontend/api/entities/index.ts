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

export type Passkey = {
  id: string;
  name: string;
};
