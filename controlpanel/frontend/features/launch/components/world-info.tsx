import React, {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List, Skeleton} from '@mui/material';

import {APIError, getWorldInfo} from '@/api';
import {WorldInfo as WorldInfoEntity} from '@/api/entities';
import {CopyableListItem} from '@/components';

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
        <CopyableListItem key="game_version" title={t('world_info_game_version')}>
          {worldInfo ? worldInfo.version : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="world_name" title={t('world_info_world_name')}>
          {worldInfo ? worldInfo.worldName.replace(/^[0-9]+-/, '') : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
        <CopyableListItem key="seed" title={t('world_info_seed')}>
          {worldInfo ? worldInfo.seed : <Skeleton animation="wave" height={24} width="25%" />}
        </CopyableListItem>
      </List>
    </Box>
  );
};

export default WorldInfo;