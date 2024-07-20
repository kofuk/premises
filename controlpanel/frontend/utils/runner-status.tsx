import {createContext, useContext, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {StatusExtraData as ExtraData} from '@/api/entities';

type RunnerStatusContextType = {
  statusCode: number;
  message: string;
  extra: ExtraData;
  pageCode: number;
  updateStatus: (statusCode: number, extra?: ExtraData, pageCode?: number) => void;
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
    }
  };

  return <RunnerStatusContext.Provider value={value}>{children}</RunnerStatusContext.Provider>;
};

export const useRunnerStatus = () => {
  return useContext(RunnerStatusContext);
};
