import React, {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List, Skeleton} from '@mui/material';

import {APIError, getSystemInfo} from '@/api';
import {SystemInfo as SystemInfoEntity} from '@/api/entities';
import {CopyableListItem} from '@/components';

const SystemInfo = () => {
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
    <Box>
      <List disablePadding>
        <CopyableListItem title={t('system_info_ip_addr')}>
          {systemInfo ? systemInfo.ipAddr || 'unknown' : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
        <CopyableListItem title={t('system_info_host_os')}>
          {systemInfo ? systemInfo.hostOs : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
        <CopyableListItem title={t('system_info_server_version')}>
          {systemInfo ? systemInfo.premisesVersion : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default SystemInfo;
