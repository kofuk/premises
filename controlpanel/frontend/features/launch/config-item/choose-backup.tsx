import React, {useEffect} from 'react';

import {useTranslation} from 'react-i18next';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

import {useBackups} from '@/api';
import Loading from '@/components/loading';

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

  const {backups, isLoading} = useBackups();
  useEffect(() => {
    if (backups && backups.length > 0) {
      if (worldName === '') {
        setWorldName(backups[0].worldName);
      }
    }
  }, [backups]);

  const handleChangeWorld = (worldName: string) => {
    const generations = backups?.find((e) => e.worldName === worldName)!.generations;
    if (generations) {
      setWorldName(worldName);
      setBackupGeneration(generations[0].id);
    }
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
          {backups?.map((e) => (
            <option value={e.worldName} key={e.worldName}>
              {e.worldName.replace(/^[0-9]+-/, '')}
            </option>
          ))}
        </select>
      </div>
    );
    const worldData = backups!.find((e) => e.worldName === worldName);
    const generations = worldData && (
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

  const content = !backups || backups?.length === 0 ? createEmptyMessage() : createBackupSelector();

  return (
    <ConfigContainer title={t('config_choose_backup')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      {(isLoading && <Loading compact />) || (
        <>
          {content}
          <div className="m-1 text-end">
            <button type="button" className="btn btn-primary" onClick={nextStep} disabled={backups?.length === 0}>
              {t('next')}
            </button>
          </div>
        </>
      )}
    </ConfigContainer>
  );
};

export default ChooseBackup;
