import {createContext, useContext, useState} from 'react';

import {useTranslation} from 'react-i18next';

import type {StatusExtraData as ExtraData} from '@/api/entities';

type RunnerStatusContextType = {
  statusCode: number;
  message: string;
  extra: ExtraData;
  pageCode: number;
  updateStatus: (statusCode: number, extra?: ExtraData, pageCode?: number) => void;
  updateCpuUsage: (usage: {cpuUsage: number; time: number}) => void;
  cpuUsage: {cpuUsage: number; time: number}[];
};

const RunnerStatusContext = createContext<RunnerStatusContextType>(null!);

export const RunnerStatusProvider = ({children}: {children: React.ReactNode}) => {
  const [t] = useTranslation();

  const [statusCode, setStatusCode] = useState(0);
  const [extra, setExtra] = useState<ExtraData>({
    progress: 0,
    textData: ''
  });
  const [pageCode, setPageCode] = useState(1);
  const [cpuUsage, setCpuUsage] = useState(
    [...Array(100)].map((_) => {
      return {cpuUsage: 0, time: 0};
    })
  );

  const value = {
    statusCode,
    message: t(`status.code_${statusCode}`),
    extra,
    pageCode,
    updateStatus: (statusCode: number, extra?: ExtraData, pageCode?: number) => {
      setStatusCode(statusCode);
      setExtra(extra || {progress: 0, textData: ''});
      if (typeof pageCode === 'number') {
        setPageCode(pageCode);
      }
    },
    updateCpuUsage: (event: {cpuUsage: number; time: number}) => {
      setCpuUsage((current) => [...current.slice(1, 100), event]);
    },
    cpuUsage
  };

  return <RunnerStatusContext.Provider value={value}>{children}</RunnerStatusContext.Provider>;
};

export const useRunnerStatus = () => {
  return useContext(RunnerStatusContext);
};
