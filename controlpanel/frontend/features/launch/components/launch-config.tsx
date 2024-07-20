import {ReactNode, createContext, useContext} from 'react';

import useSWR from 'swr';

import {launch as apiLaunch, reconfigure as apiReconfigure, updateConfig as apiUpdateConfig, getConfig} from '@/api';
import {PendingConfig} from '@/api/entities';
import Loading from '@/components/loading';

type ConfigContextType = {
  config: PendingConfig;
  updateConfig: (config: PendingConfig) => Promise<void>;
  isValid: boolean;
  launch: () => Promise<void>;
  reconfigure: () => Promise<void>;
};

const ConfigContext = createContext<ConfigContextType>(null!);

export const ConfigProvider = ({children}: {children: ReactNode}) => {
  const {data, mutate, error, isLoading} = useSWR('/api/config', async () => await getConfig());
  if (isLoading || error) {
    // TODO: Proper error handling
    return <Loading />;
  }

  const {isValid, config: remoteConfig} = data!;

  const updateConfig = async (config: PendingConfig): Promise<void> => {
    await mutate(apiUpdateConfig(config));
  };

  const launch = async (): Promise<void> => {
    await apiLaunch();
  };

  const reconfigure = async (): Promise<void> => {
    await apiReconfigure();
  };

  const value = {
    config: remoteConfig!,
    updateConfig,
    isValid,
    launch,
    reconfigure
  };

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
};

export const useLaunchConfig = () => {
  return useContext(ConfigContext);
};
