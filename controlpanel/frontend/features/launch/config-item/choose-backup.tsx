import React, {useEffect, useState} from 'react';

import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';
import {useTranslation} from 'react-i18next';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';
import {WorldBackup} from './world-backup';

type Props = ItemProp & {
  worldName: string;
  backupGeneration: string;
  useCachedWorld: boolean;
  setWorldName: (val: string) => void;
  setBackupGeneration: (val: string) => void;
  setUseCachedWorld: (val: boolean) => void;
};

const ChooseBackup = ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  worldName,
  backupGeneration,
  setWorldName,
  setBackupGeneration,
  useCachedWorld,
  setUseCachedWorld
}: Props) => {
  const [t] = useTranslation();

  const [backups, setBackups] = useState<WorldBackup[]>([]);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    refreshBackups();
  }, []);

  const refreshBackups = (reload: boolean = false) => {
    (async () => {
      setRefreshing(true);
      try {
        const backups = await fetch(`/api/backups${reload ? '?reload' : ''}`).then((resp) => resp.json());
        setBackups(backups);
        if (backups.length > 0) {
          setWorldName(backups[0].worldName);
          setBackupGeneration(backups[0].generations[0].id);
        }
      } catch (err) {
        console.error(err);
      } finally {
        setRefreshing(false);
      }
    })();
  };

  const handleChangeWorld = (worldName: string) => {
    setWorldName(worldName);
    const generations = backups.find((e) => e.worldName === worldName)!.generations;
    setBackupGeneration(generations[0].id);
  };

  const handleChangeGeneration = (generationId: string) => {
    setBackupGeneration(generationId);
  };

  const createBackupSelector = (): React.ReactElement => {
    const worlds = (
      <div className="m-2">
        <label className="form-label" htmlFor="worldSelect">
          {t('select_world')}
        </label>
        <select className="form-select" value={worldName} id="worldSelect" onChange={(e) => handleChangeWorld(e.target.value)}>
          {backups.map((e) => (
            <option value={e.worldName} key={e.worldName}>
              {e.worldName.replace(/^[0-9]+-/, '')}
            </option>
          ))}
        </select>
      </div>
    );
    const worldData = backups.find((e) => e.worldName === worldName);
    const generations = worldData ? (
      <div className="m-2">
        <label className="form-label" htmlFor="backupGenerationSelect">
          {t('backup_generation')}
        </label>
        <select className="form-select" value={backupGeneration} id="backupGenerationSelect" onChange={(e) => handleChangeGeneration(e.target.value)}>
          {worldData.generations.map((e) => {
            const dateTime = new Date(e.timestamp);
            return (
              <option value={e.id} key={e.gen}>
                {(e.gen == 'latest' ? 'Latest' : `${e.gen} gen ago`) + ` (${dateTime.toLocaleString()})`}
              </option>
            );
          })}
        </select>
      </div>
    ) : (
      <></>
    );

    return (
      <>
        {worlds}
        {generations}
        <div className="m-2 form-check form-switch">
          <input
            className="form-check-input"
            type="checkbox"
            id="useCachedWorld"
            checked={useCachedWorld}
            onChange={(e) => setUseCachedWorld(e.target.checked)}
          />
          <label className="form-check-label" htmlFor="useCachedWorld">
            {t('use_cached_world')}
          </label>
        </div>
        <div className="m-1">
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={() => refreshBackups(true)} disabled={refreshing}>
            {refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
            {t('refresh')}
          </button>
        </div>
      </>
    );
  };

  const createEmptyMessage = (): React.ReactElement => {
    return (
      <div className="alert alert-warning" role="alert">
        {t('no_backups')}
      </div>
    );
  };

  const content = backups.length === 0 ? createEmptyMessage() : createBackupSelector();

  return (
    <ConfigContainer title={t('config_choose_backup')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      {content}
      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep} disabled={backups.length === 0}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default ChooseBackup;
