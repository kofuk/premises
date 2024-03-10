import React, {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List} from '@mui/material';

import {APIError, getSystemInfo} from '@/api';
import {SystemInfo as SystemInfoEntity} from '@/api/entities';
import {CopyableListItem, DelayedSkeleton} from '@/components';

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
          {systemInfo ? (
            systemInfo.ipAddr || 'unknown'
          ) : (
            <DelayedSkeleton width="25%">
              <Box sx={{opacity: 0}}>-</Box>
            </DelayedSkeleton>
          )}
        </CopyableListItem>
        <CopyableListItem title={t('system_info_host_os')}>
          {systemInfo ? (
            systemInfo.hostOs
          ) : (
            <DelayedSkeleton width="25%">
              <Box sx={{opacity: 0}}>-</Box>
            </DelayedSkeleton>
          )}
        </CopyableListItem>
        <CopyableListItem title={t('system_info_server_version')}>
          {systemInfo ? (
            systemInfo.premisesVersion
          ) : (
            <DelayedSkeleton width="25%">
              <Box sx={{opacity: 0}}>-</Box>
            </DelayedSkeleton>
          )}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default SystemInfo;
