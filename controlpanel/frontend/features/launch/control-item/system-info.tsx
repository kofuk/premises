import React, {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';
import {List, Skeleton} from '@mui/material';

import {APIError, getSystemInfo} from '@/api';
import {SystemInfo as SystemInfoEntity} from '@/api/entities';
import {CopyableListItem} from '@/components';

type Prop = {
  backToMenu: () => void;
};

const SystemInfo = ({backToMenu}: Prop) => {
  const [t] = useTranslation();

  const [systemInfo, setSystemInfo] = useState<SystemInfoEntity | null>(null);
  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    (async () => {
      try {
        setSystemInfo(await getSystemInfo());
      } catch (err) {
        if (err instanceof APIError) {
          enqueueSnackbar(err.message, {variant: 'error'});
        }
      }
    })();
  }, []);

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <ArrowBackIcon /> {t('back')}
      </button>
      <div className="m-2">
        {' '}
        <List disablePadding>
          <CopyableListItem key="server_version" title={t('system_info_server_version')}>
            {systemInfo ? systemInfo.premisesVersion : <Skeleton animation="wave" height={24} width="25%" />}
          </CopyableListItem>
          <CopyableListItem key="host_os" title={t('system_info_host_os')}>
            {systemInfo ? systemInfo.hostOs : <Skeleton animation="wave" height={24} width="25%" />}
          </CopyableListItem>
        </List>
      </div>
    </div>
  );
};

export default SystemInfo;
