import React, {ReactNode, useContext, useEffect, useState} from 'react';

import {launch as apiLaunch, reconfigure as apiReconfigure, updateConfig as apiUpdateConfig, createConfig} from '@/api';
import {PendingConfig} from '@/api/entities';
import {Loading} from '@/components';

type ConfigContextType = {
  config: PendingConfig;
  updateConfig: (config: PendingConfig) => Promise<void>;
  isValid: boolean;
  launch: () => Promise<void>;
  reconfigure: () => Promise<void>;
};

const ConfigContext = React.createContext<ConfigContextType>(null!);

export const ConfigProvider = ({children}: {children: ReactNode}) => {
  const [remoteConfig, setRemoteConfig] = useState<PendingConfig | null>(null);
  const [isValid, setIsValid] = useState(false);
  useEffect(() => {
    (async () => {
      let configShareId = null;

      const hash = location.hash;

      if (hash && hash.length > 1) {
        const params = new URLSearchParams(hash.substr(1));
        configShareId = params.get('configShareId');
      }

      const {isValid, config: newConfig} = await createConfig({configShareId});
      setRemoteConfig(newConfig);
      setIsValid(isValid);
    })();
  }, []);

  if (remoteConfig === null) {
    return <Loading />;
  }

  const updateConfig = async (config: PendingConfig): Promise<void> => {
    config.id = remoteConfig!.id;
    const {isValid, config: newConfig} = await apiUpdateConfig(config);
    setRemoteConfig(newConfig);
    setIsValid(isValid);
  };

  const launch = async (): Promise<void> => {
    await apiLaunch({id: remoteConfig!.id!});
  };

  const reconfigure = async (): Promise<void> => {
    await apiReconfigure({id: remoteConfig!.id!});
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
