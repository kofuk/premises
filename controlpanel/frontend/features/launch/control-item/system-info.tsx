import React, {useState, useEffect} from 'react';

import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';
import {useTranslation} from 'react-i18next';

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
      <div className="list-group">
        <CopyableListItem title={t('system_info_server_version')} content={systemInfo.premisesVersion} />
        <CopyableListItem title={t('system_info_host_os')} content={systemInfo.hostOS} />
      </div>
    );
  }

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <IoIosArrowBack /> {t('back')}
      </button>
      <div className="m-2">{mainContents}</div>
    </div>
  );
};

export default SystemInfo;
