import React, {ReactNode, useContext, useEffect, useState} from 'react';

import {launch as apiLaunch, reconfigure as apiReconfigure, updateConfig as apiUpdateConfig, createConfig} from '@/api';
import {PendingConfig} from '@/api/entities';
import {Loading} from '@/components';

type ConfigContextType = {
  configId: string;
  updateConfig: (config: PendingConfig) => Promise<void>;
  launch: () => Promise<void>;
  reconfigure: () => Promise<void>;
};

const ConfigContext = React.createContext<ConfigContextType>(null!);

export const ConfigProvider = ({children}: {children: ReactNode}) => {
  const [configId, setConfigId] = useState<string | null>(null);
  useEffect(() => {
    (async () => {
      let configShareId = null;

      const hash = location.hash;

      if (hash && hash.length > 1) {
        const params = new URLSearchParams(hash.substr(1));
        configShareId = params.get('configShareId');
        location.hash = '';
      }

      const data = await createConfig({configShareId});
      setConfigId(data.id!);
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
    configId: configId!,
    updateConfig,
    launch,
    reconfigure
  };

  return <ConfigContext.Provider value={value}>{children}</ConfigContext.Provider>;
};

export const useLaunchConfig = () => {
  return useContext(ConfigContext);
};
