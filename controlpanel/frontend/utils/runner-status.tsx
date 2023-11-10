import React, {useContext, useState} from 'react';

import {useTranslation} from 'react-i18next';

type RunnerStatusContextType = {
  statusCode: number;
  message: string;
  progress: number;
  pageCode: number;
  updateStatus: (statusCode: number, progress: number, pageCode?: number) => void;
};

const RunnerStatusContext = React.createContext<RunnerStatusContextType>(null!);

export const RunnerStatusProvider = ({children}: {children: React.ReactNode}) => {
  const [t] = useTranslation();

  const [statusCode, setStatusCode] = useState(0);
  const [progress, setProgress] = useState(0);
  const [pageCode, setPageCode] = useState(1);

  const value = {
    statusCode,
    message: t(`status.code_${statusCode}`),
    progress,
    pageCode,
    updateStatus: (statusCode: number, progress: number, pageCode?: number) => {
      setStatusCode(statusCode);
      setProgress(progress);
      if (typeof pageCode === 'number') {
        setPageCode(pageCode);
      }
    }
  };

  return <RunnerStatusContext.Provider value={value}>{children}</RunnerStatusContext.Provider>;
};

export const useRunnerStatus = () => {
  return useContext(RunnerStatusContext);
};
