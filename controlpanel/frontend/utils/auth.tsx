import React, {ReactNode, useState, useEffect, useContext} from 'react';
import {encodeBuffer, decodeBuffer} from '@/utils/base64url';

export enum LoginResult {
  LoggedIn,
  NeedsChangePassword
}

type AuthContextType = {
  loggedIn: boolean;
  login: (username: string, password: string) => Promise<LoginResult>;
  loginWebAuthn: (username: string) => Promise<void>;
  logout: () => Promise<void>;
  initializePassword: (username: string, newPassword: string) => Promise<void>;
};

const AuthContext = React.createContext<AuthContextType>(null!);

export const AuthProvider = ({children}: {children: ReactNode}) => {
  let [loggedIn, setLoggedIn] = useState(false);
  let [initialized, setInitialized] = useState(false);
  useEffect(() => {
    (async () => {
      const loggedIn = await fetch('/api/current-user')
        .then((resp) => resp.json())
        .then((resp) => resp['success'] as boolean);
      setLoggedIn(loggedIn);
      setInitialized(true);
    })();
  }, []);

  if (!initialized) {
    return <div>ohyohyoi♪～(´ε｀ )</div>;
  }

  const login = async (username: string, password: string): Promise<LoginResult> => {
    const params = new URLSearchParams();
    params.append('username', username);
    params.append('password', password);

    const resp = await fetch('/login', {
      method: 'post',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: params.toString()
    }).then((resp) => resp.json());

    if (!resp['success']) {
      throw new Error(resp['reason']);
    }

    if (resp['needsChangePassword']) {
      return LoginResult.NeedsChangePassword;
    }

    setLoggedIn(true);

    return LoginResult.LoggedIn;
  };

  const loginWebAuthn = async (username: string): Promise<void> => {
    const params = new URLSearchParams();
    params.append('username', username);

    let beginResp: any = await fetch('/login/hardwarekey/begin', {
      method: 'post',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: params.toString()
    }).then((resp) => resp.json());

    if (!beginResp['success']) {
      throw new Error(beginResp['reason']);
    }

    const options = beginResp['options'];

    options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
    if (options.publicKey.allowCredentials) {
      for (let i = 0; i < options.publicKey.allowCredentials.length; i++) {
        options.publicKey.allowCredentials[i].id = decodeBuffer(options.publicKey.allowCredentials[i].id);
      }
    }

    const publicKeyCred = (await navigator.credentials.get(options)) as PublicKeyCredential;
    const rawId = publicKeyCred.rawId;
    const {authenticatorData, clientDataJSON, signature, userHandle} = publicKeyCred.response as AuthenticatorAssertionResponse;

    const finishResp: any = await fetch('/login/hardwarekey/finish', {
      method: 'post',
      body: JSON.stringify({
        id: publicKeyCred.id,
        rawId: encodeBuffer(rawId),
        type: publicKeyCred.type,
        response: {
          authenticatorData: encodeBuffer(authenticatorData),
          clientDataJSON: encodeBuffer(clientDataJSON),
          signature: encodeBuffer(signature),
          userHandle: encodeBuffer(userHandle!!)
        }
      })
    }).then((resp) => resp.json());

    if (!finishResp['success']) {
      throw new Error(finishResp['reason']);
    }

    setLoggedIn(true);
  };

  const logout = async () => {
    const resp = await fetch('/logout', {method: 'POST'}).then((resp) => resp.json());
    if (!resp['success']) {
      throw new Error(resp['reason']);
    }
    setLoggedIn(false);
  };

  const initializePassword = async (username: string, newPassword: string): Promise<void> => {
    const params = new URLSearchParams();
    params.append('username', username);
    params.append('password', newPassword);

    const resp = await fetch('/login/reset-password', {
      method: 'post',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: params.toString()
    }).then((resp) => resp.json());

    if (!resp['success']) {
      throw new Error(resp['reason']);
    }

    setLoggedIn(true);
  };

  const value = {
    loggedIn,
    login,
    loginWebAuthn,
    logout,
    initializePassword
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  return useContext(AuthContext);
};
