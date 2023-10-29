import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {useBackups} from '@/api';
import {Loading} from '@/components';
import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

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

  const {data: backups, isLoading} = useBackups();
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
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_world_name')}>
      {(isLoading && <Loading compact />) || (
        <>
          <label className="form-label" htmlFor="newWorldName">
            {t('world_name')}
          </label>
          <input
            className="form-control"
            id="newWorldName"
            onChange={(e) => {
              handleChange(e.target.value);
            }}
            type="text"
            value={worldName}
          />
          {alert}
        </>
      )}
      <div className="m-1 text-end">
        <button className="btn btn-primary" disabled={worldName.length === 0 || duplicateName || invalidName} onClick={nextStep} type="button">
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default WorldName;
