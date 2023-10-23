import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';
import useSWR from 'swr';

import {getBackups} from '@/api';
import Loading from '@/components/loading';
import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';


const useBackups = () => {
  const {data, isLoading} = useSWR('/api/backups', getBackups);
  return {
    backups: data,
    isLoading
  };
};

const WorldName = ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  worldName,
  setWorldName
}: ItemProp & {
  worldName: string;
  setWorldName: (val: string) => void;
}) => {
  const [t] = useTranslation();

  const {backups, isLoading} = useBackups();
  const [duplicateName, setDuplicateName] = useState(false);
  const [invalidName, setInvalidName] = useState(false);

  const handleChange = (val: string) => {
    setWorldName(val);

    if (!val.match(/^[- _a-zA-Z0-9()]+$/)) {
      setInvalidName(true);
      return;
    }
    if (backups?.find((e) => e.worldName === val)) {
      setDuplicateName(true);
      return;
    }

    setDuplicateName(false);
    setInvalidName(false);
  };

  let alert = <></>;
  if (invalidName) {
    alert = (
      <div className="m-2 alert alert-danger" role="alert">
        Name must be alphanumeric.
      </div>
    );
  } else if (duplicateName) {
    alert = (
      <div className="m-2 alert alert-danger" role="alert">
        World name duplicates.
      </div>
    );
  }

  return (
    <ConfigContainer title={t('config_world_name')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      {(isLoading && <Loading compact />) || (
        <>
          <label className="form-label" htmlFor="newWorldName">
            {t('world_name')}
          </label>
          <input
            type="text"
            className="form-control"
            id="newWorldName"
            value={worldName}
            onChange={(e) => {
              handleChange(e.target.value);
            }}
          />
          {alert}
        </>
      )}
      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep} disabled={worldName.length === 0 || duplicateName || invalidName}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default WorldName;
