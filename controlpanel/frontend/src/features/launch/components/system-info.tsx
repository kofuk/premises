import {Box, List} from '@mui/material';
import {useEffect, useState} from 'react';
import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

import {APIError, getSystemInfo} from '@/api';
import type {SystemInfo as SystemInfoEntity} from '@/api/entities';
import CopyableListItem from '@/components/copyable-list-item';
import DelayedSkeleton from '@/components/delayed-skeleton';
import {useAuth} from '@/utils/auth';

const SystemInfo = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const [systemInfo, setSystemInfo] = useState<SystemInfoEntity | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setSystemInfo(await getSystemInfo(accessToken));
      } catch (err) {
        if (err instanceof APIError) {
          toast.error(err.message);
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
