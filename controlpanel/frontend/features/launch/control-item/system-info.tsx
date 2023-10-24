import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';
import {List} from '@mui/material';

import CopyableListItem from '@/components/copyable-list-item';

type Prop = {
  backToMenu: () => void;
};

type SystemInfoData = {
  premisesVersion: string;
  hostOS: string;
} | null;

const SystemInfo = ({backToMenu}: Prop) => {
  const [t] = useTranslation();

  const [systemInfo, setSystemInfo] = useState<SystemInfoData | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const systemInfo = await fetch('/api/systeminfo').then((resp) => resp.json());
        setSystemInfo(systemInfo);
      } catch (err) {
        console.error(err);
      }
    })();
  }, []);

  let mainContents: React.ReactElement;
  if (systemInfo === null) {
    mainContents = <></>;
  } else {
    mainContents = (
      <List disablePadding>
        <CopyableListItem key="server_version" title={t('system_info_server_version')}>
          {systemInfo.premisesVersion}
        </CopyableListItem>
        <CopyableListItem key="host_os" title={t('system_info_host_os')}>
          {systemInfo.hostOS}
        </CopyableListItem>
      </List>
    );
  }

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <ArrowBackIcon /> {t('back')}
      </button>
      <div className="m-2">{mainContents}</div>
    </div>
  );
};

export default SystemInfo;
