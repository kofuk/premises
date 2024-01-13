import React, {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

import {APIError, useBackups} from '@/api';
import {Loading} from '@/components';

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

  const {enqueueSnackbar} = useSnackbar();

  const {data: backups, error, isLoading} = useBackups();
  useEffect(() => {
    if (backups && backups.length > 0) {
      if (worldName === '') {
        setWorldName(backups[0].worldName);
        setBackupGeneration(backups[0].generations[0].id);
      }
    }
  }, [backups]);
  useEffect(() => {
    if (error) {
      if (error instanceof APIError) {
        enqueueSnackbar(error.message, {variant: 'error'});
      }
    }
  }, [error]);

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
        <select className="form-select" id="worldSelect" onChange={(e) => handleChangeWorld(e.target.value)} value={worldName}>
          {backups?.map((e) => (
            <option key={e.worldName} value={e.worldName}>
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
        <select className="form-select" id="backupGenerationSelect" onChange={(e) => handleChangeGeneration(e.target.value)} value={backupGeneration}>
          {worldData.generations.map((e) => {
            const dateTime = new Date(e.timestamp);
            const label = e.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
              ? dateTime.toLocaleString()
              : `${e.gen} (${dateTime.toLocaleString()})`;
            return (
              <option key={e.gen} value={e.id}>
                {label}
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
            checked={useCachedWorld}
            className="form-check-input"
            id="useCachedWorld"
            onChange={(e) => setUseCachedWorld(e.target.checked)}
            type="checkbox"
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
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_choose_backup')}>
      {isLoading ? (
        <Loading compact />
      ) : (
        <>
          {content}
          <div className="m-1 text-end">
            <button className="btn btn-primary" disabled={backups?.length === 0} onClick={nextStep} type="button">
              {t('next')}
            </button>
          </div>
        </>
      )}
    </ConfigContainer>
  );
};

export default ChooseBackup;
