import {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List} from '@mui/material';

import {APIError, getWorldInfo} from '@/api';
import {WorldInfo as WorldInfoEntity} from '@/api/entities';
import CopyableListItem from '@/components/copyable-list-item';
import DelayedSkeleton from '@/components/delayed-skeleton';

const WorldInfo = () => {
  const [t] = useTranslation();

  const [worldInfo, setWorldInfo] = useState<WorldInfoEntity | null>(null);
  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    (async () => {
      try {
        setWorldInfo(await getWorldInfo());
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
        <CopyableListItem key="game_version" title={t('launch.world_info.game_version')}>
          {worldInfo ? worldInfo.version : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="world_name" title={t('launch.world_info.world_name')}>
          {worldInfo ? worldInfo.worldName.replace(/^[0-9]+-/, '') : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="seed" title={t('launch.world_info.seed')}>
          {worldInfo ? worldInfo.seed : <DelayedSkeleton width="25%" />}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default WorldInfo;
