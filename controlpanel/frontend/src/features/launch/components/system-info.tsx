import {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List} from '@mui/material';

import {APIError, getSystemInfo} from '@/api';
import {SystemInfo as SystemInfoEntity} from '@/api/entities';
import CopyableListItem from '@/components/copyable-list-item';
import DelayedSkeleton from '@/components/delayed-skeleton';
import {useAuth} from '@/utils/auth';

const SystemInfo = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const [systemInfo, setSystemInfo] = useState<SystemInfoEntity | null>(null);
  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    (async () => {
      try {
        setSystemInfo(await getSystemInfo(accessToken));
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
        <CopyableListItem title={t('launch.system_info.host_os')}>
          {systemInfo ? systemInfo.hostOs : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
        <CopyableListItem title={t('launch.system_info.runner_build')}>
          {systemInfo ? systemInfo.premisesVersion : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default SystemInfo;
