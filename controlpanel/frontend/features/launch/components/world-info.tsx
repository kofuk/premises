import React, {useEffect, useState} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {Box, List} from '@mui/material';

import {APIError, getWorldInfo} from '@/api';
import {WorldInfo as WorldInfoEntity} from '@/api/entities';
import {CopyableListItem, DelayedSkeleton} from '@/components';

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
          {worldInfo ? (
            worldInfo.version
          ) : (
            <DelayedSkeleton width="25%">
              <Box sx={{opacity: 0}}>-</Box>
            </DelayedSkeleton>
          )}
        </CopyableListItem>
        <CopyableListItem key="world_name" title={t('world_info_world_name')}>
          {worldInfo ? (
            worldInfo.worldName.replace(/^[0-9]+-/, '')
          ) : (
            <DelayedSkeleton width="25%">
              <Box sx={{opacity: 0}}>-</Box>
            </DelayedSkeleton>
          )}
        </CopyableListItem>
        <CopyableListItem key="seed" title={t('world_info_seed')}>
          {worldInfo ? (
            worldInfo.seed
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

export default WorldInfo;
