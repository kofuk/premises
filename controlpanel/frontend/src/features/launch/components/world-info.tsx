import {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

import {Box, List} from '@mui/material';

import {APIError, getWorldInfo} from '@/api';
import {WorldInfo as WorldInfoEntity} from '@/api/entities';
import CopyableListItem from '@/components/copyable-list-item';
import DelayedSkeleton from '@/components/delayed-skeleton';
import {useAuth} from '@/utils/auth';

const WorldInfo = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const [worldInfo, setWorldInfo] = useState<WorldInfoEntity | null>(null);

  useEffect(() => {
    (async () => {
      try {
        setWorldInfo(await getWorldInfo(accessToken));
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
        <CopyableListItem key="game_version" title={t('launch.world_info.game_version')}>
          {worldInfo ? worldInfo.version : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="world_name" title={t('launch.world_info.world_name')}>
          {worldInfo ? worldInfo.worldName : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="seed" title={t('launch.world_info.seed')}>
          {worldInfo ? worldInfo.seed : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default WorldInfo;
