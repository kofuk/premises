import {useState, useEffect} from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

import '../../i18n';
import {t} from 'i18next';

import CopyableListItem from '../../components/copyable-list-item';

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

export default (props: Prop) => {
  const {backToMenu} = props;

  const [worldInfo, setWorldInfo] = useState<WorldInfoData | null>(null);
  const [refreshing, setRefreshing] = useState(true);

  useEffect(() => {
    fetch('/api/worldinfo')
      .then((resp) => resp.json())
      .then((resp) => {
        setWorldInfo(resp);
        setRefreshing(false);
      });
  }, []);

  const handleRefresh = () => {
    setRefreshing(true);
    fetch('/api/worldinfo')
      .then((resp) => resp.json())
      .then((resp) => {
        setWorldInfo(resp);
        setRefreshing(false);
      });
  };

  let mainContents: React.ReactElement;
  if (worldInfo === null) {
    mainContents = <></>;
  } else {
    mainContents = (
      <div className="list-group">
        <CopyableListItem title={t('world_info_game_version')} content={worldInfo.serverVersion} />
        <CopyableListItem title={t('world_info_world_name')} content={worldInfo.world.name.replace(/^[0-9]+-/, '')} />
        <CopyableListItem title={t('world_info_seed')} content={worldInfo.world.seed} />
      </div>
    );
  }

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <IoIosArrowBack /> {t('back')}
      </button>
      <div className="m-2">{mainContents}</div>
      <div className="m-1">
        <button type="button" className="btn btn-sm btn-outline-secondary" onClick={handleRefresh} disabled={refreshing}>
          {refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
          {t('refresh')}
        </button>
      </div>
    </div>
  );
};
