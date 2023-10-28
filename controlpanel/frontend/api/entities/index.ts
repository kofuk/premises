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

export type CredentialNameAndCreationResponse = {
  name: string;
  credentialCreationResponse: {
    id: string;
    rawId: string;
    type: string;
    response: {
      attestationObject: string;
      clientDataJSON: string;
    };
  };
};

export type CredentialAssertionResponse = {
  id: string;
  rawId: string;
  type: string;
  response: {
    authenticatorData: string;
    clientDataJSON: string;
    signature: string;
    userHandle: string;
  };
};
