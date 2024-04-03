import React, {ReactNode, useContext, useEffect, useState} from 'react';

import {launch as apiLaunch, reconfigure as apiReconfigure, updateConfig as apiUpdateConfig, createConfig} from '@/api';
import {PendingConfig} from '@/api/entities';
import {Loading} from '@/components';

type ConfigContextType = {
  updateConfig: (config: PendingConfig) => Promise<void>;
  launch: () => Promise<void>;
  reconfigure: () => Promise<void>;
};

const ConfigContext = React.createContext<ConfigContextType>(null!);

export const ConfigProvider = ({children}: {children: ReactNode}) => {
  const [configId, setConfigId] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      const data = await createConfig();
      setConfigId(data.id);
    })();
  }, []);

  if (configId === null) {
    return <Loading />;
  }

  const updateConfig = async (config: PendingConfig): Promise<void> => {
    config.id = configId;
    await apiUpdateConfig(config);
  };

  const launch = async (): Promise<void> => {
    await apiLaunch({id: configId as string});
  };

  const reconfigure = async (): Promise<void> => {
    await apiReconfigure({id: configId as string});
  };

  const value = {
    updateConfig,
    launch,
    reconfigure
  };

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
};

export const useLaunchConfig = () => {
  return useContext(ConfigContext);
};
