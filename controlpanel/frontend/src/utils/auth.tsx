import {ReactNode, createContext, useContext} from 'react';

import {login as apiLogin, useSessionData} from '@/api';
import Loading from '@/components/loading';

export enum LoginResult {
  LoggedIn,
  NeedsChangePassword
}

type AuthContextType = {
  loggedIn: boolean;
  userName: string | null;
  login: (userName: string, password: string) => Promise<LoginResult>;
  logout: () => Promise<void>;
  initializePassword: (username: string, newPassword: string) => Promise<void>;
};

const AuthContext = createContext<AuthContextType>(null!);

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
    logout,
    initializePassword
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};

export const useAuth = () => {
  return useContext(AuthContext);
};
