import React, {ReactNode, useContext, useEffect, useState} from 'react';

import {login as apiLogin, useSessionData} from '@/api';
import Loading from '@/components/loading';
import {decodeBuffer, encodeBuffer} from '@/utils/base64url';

export enum LoginResult {
  LoggedIn,
  NeedsChangePassword
}

type AuthContextType = {
  loggedIn: boolean;
  userName: string | null;
  login: (username: string, password: string) => Promise<LoginResult>;
  loginPasskeys: () => Promise<void>;
  logout: () => Promise<void>;
  initializePassword: (username: string, newPassword: string) => Promise<void>;
};

const AuthContext = React.createContext<AuthContextType>(null!);

export const AuthProvider = ({children}: {children: ReactNode}) => {
  const {data: session, error, isLoading, mutate} = useSessionData();
  if (isLoading) {
    return <Loading />;
  }
  if (error) {
    throw error;
  }

  const login = async (username: string, password: string): Promise<LoginResult> => {
    const resp = await apiLogin({username, password});
    if (resp.needsChangePassword) {
      return LoginResult.NeedsChangePassword;
    }

    mutate();

    return LoginResult.LoggedIn;
  };

  const loginPasskeys = async (): Promise<void> => {
    const beginResp: any = await fetch('/login/hardwarekey/begin', {
      method: 'post',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      }
    }).then((resp) => resp.json());

    if (!beginResp['success']) {
      throw new Error(beginResp['reason']);
    }

    const options = beginResp['options'];

    options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
    options.publicKey.allowCredentials = [];
    options.mediation = 'conditional';

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
          userHandle: encodeBuffer(userHandle ?? new ArrayBuffer(0))
        }
      })
    }).then((resp) => resp.json());

    if (!finishResp['success']) {
      throw new Error(finishResp['reason']);
    }

    mutate();
  };

  const logout = async () => {
    const resp = await fetch('/logout', {method: 'POST'}).then((resp) => resp.json());
    if (!resp['success']) {
      throw new Error(resp['reason']);
    }
    mutate();
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

    mutate();
  };

  const value = {
    loggedIn: !!session && session.loggedIn,
    userName: (session && session.userName) || null,
    login,
    loginPasskeys,
    logout,
    initializePassword
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  return useContext(AuthContext);
};

const passkeysSupported = async (): Promise<boolean> => {
  try {
    const supported = (
      await Promise.all([PublicKeyCredential.isConditionalMediationAvailable(), PublicKeyCredential.isUserVerifyingPlatformAuthenticatorAvailable()])
    ).every((v) => v);

    return supported;
  } catch (_) {
    return false;
  }
};

export const usePasskeysSupported = (): boolean => {
  const [supported, setSupported] = useState(false);
  useEffect(() => {
    (async () => {
      setSupported(await passkeysSupported());
    })();
  }, []);

  return supported;
};
