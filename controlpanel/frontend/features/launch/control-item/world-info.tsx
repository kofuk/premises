import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon, Refresh as RefreshIcon} from '@mui/icons-material';
import {List} from '@mui/material';

import {CopyableListItem} from '@/components';

type Prop = {
  backToMenu: () => void;
};

type WorldDetail = {
  name: string;
  seed: string;
};

type WorldInfoData = {
  serverVersion: string;
  world: WorldDetail;
};

const WorldInfo = (props: Prop) => {
  const [t] = useTranslation();

  const {backToMenu} = props;

  const [worldInfo, setWorldInfo] = useState<WorldInfoData | null>(null);
  const [refreshing, setRefreshing] = useState(true);

  useEffect(() => {
    refreshInfo();
  }, []);

  const refreshInfo = () => {
    (async () => {
      setRefreshing(true);
      try {
        const worldInfo = await fetch('/api/worldinfo').then((resp) => resp.json());
        setWorldInfo(worldInfo);
      } catch (err) {
        console.error(err);
      } finally {
        setRefreshing(false);
      }
    })();
  };

  let mainContents: React.ReactElement;
  if (worldInfo === null) {
    mainContents = <></>;
  } else {
    mainContents = (
      <List disablePadding>
        <CopyableListItem key="game_version" title={t('world_info_game_version')}>
          {worldInfo.serverVersion}
        </CopyableListItem>
        <CopyableListItem key="world_name" title={t('world_info_world_name')}>
          {worldInfo.world.name.replace(/^[0-9]+-/, '')}
        </CopyableListItem>
        <CopyableListItem key="seed" title={t('world_info_seed')}>
          {worldInfo.world.seed}
        </CopyableListItem>
      </List>
    );
  }

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <ArrowBackIcon /> {t('back')}
      </button>
      <div className="m-2">{mainContents}</div>
      <div className="m-1">
        <button className="btn btn-sm btn-outline-secondary" disabled={refreshing} onClick={refreshInfo} type="button">
          {refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <RefreshIcon />}
          {t('refresh')}
        </button>
      </div>
    </div>
  );
};

export default WorldInfo;
