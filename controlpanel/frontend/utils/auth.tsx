import React, {ReactNode, useContext, useEffect, useState} from 'react';

import {login as apiLogin, getPasskeysLoginOptions, loginPasskeys as loginPasskeysApi, useSessionData} from '@/api';
import {Loading} from '@/components';
import {decodeBuffer, encodeBuffer} from '@/utils/base64url';

export enum LoginResult {
  LoggedIn,
  NeedsChangePassword
}

type AuthContextType = {
  loggedIn: boolean;
  userName: string | null;
  login: (userName: string, password: string) => Promise<LoginResult>;
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

  const login = async (userName: string, password: string): Promise<LoginResult> => {
    const resp = await apiLogin({userName, password});
    if (resp.needsChangePassword) {
      return LoginResult.NeedsChangePassword;
    }

    mutate();

    return LoginResult.LoggedIn;
  };

  const loginPasskeys = async (): Promise<void> => {
    const options = await getPasskeysLoginOptions();

    options.publicKey!.challenge = decodeBuffer(options.publicKey!.challenge as unknown as string);
    options.publicKey!.allowCredentials = [];
    options.mediation = 'conditional';

    const publicKeyCred = (await navigator.credentials.get(options)) as PublicKeyCredential;
    const rawId = publicKeyCred.rawId;
    const {authenticatorData, clientDataJSON, signature, userHandle} = publicKeyCred.response as AuthenticatorAssertionResponse;

    await loginPasskeysApi({
      id: publicKeyCred.id,
      rawId: encodeBuffer(rawId),
      type: publicKeyCred.type,
      response: {
        authenticatorData: encodeBuffer(authenticatorData),
        clientDataJSON: encodeBuffer(clientDataJSON),
        signature: encodeBuffer(signature),
        userHandle: encodeBuffer(userHandle ?? new ArrayBuffer(0))
      }
    });

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
