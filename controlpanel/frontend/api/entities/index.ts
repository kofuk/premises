export interface SessionData {
  loggedIn: boolean;
  userName: string;
}

export interface GenerationInfo {
  gen: string;
  id: string;
  timestamp: number;
}

export interface WorldBackup {
  worldName: string;
  generations: GenerationInfo[];
}
