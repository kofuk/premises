import {ReactNode, createContext, useContext} from 'react';

import useSWR from 'swr';

import {launch as apiLaunch, updateConfig as apiUpdateConfig, getConfig} from '@/api';
import {PendingConfig} from '@/api/entities';
import Loading from '@/components/loading';
import {useAuth} from '@/utils/auth';

type ConfigContextType = {
  config: PendingConfig;
  updateConfig: (config: PendingConfig) => Promise<void>;
  isValid: boolean;
  launch: () => Promise<void>;
};

const ConfigContext = createContext<ConfigContextType>(null!);

export const ConfigProvider = ({children}: {children: ReactNode}) => {
  const {accessToken} = useAuth();

  const {data, mutate, error, isLoading} = useSWR('/api/config', async () => await getConfig(accessToken));
  if (isLoading || error) {
    // TODO: Proper error handling
    return <Loading />;
  }

  const {isValid, config: remoteConfig} = data!;

  const updateConfig = async (config: PendingConfig): Promise<void> => {
    await mutate(apiUpdateConfig(accessToken, config));
  };

  const launch = async (): Promise<void> => {
    await apiLaunch(accessToken);
  };

  const value = {
    config: remoteConfig!,
    updateConfig,
    isValid,
    launch
  };

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
};

export const useLaunchConfig = () => {
  return useContext(ConfigContext);
};
