import {useState, useEffect} from 'react';

import '@/i18n';
import {t} from 'i18next';

import {ItemProp} from '@/features/launch/config-item/prop';
import {WorldBackup} from '@/features/launch/config-item/world-backup';
import ConfigContainer from '@/features/launch/config-item/config-container';

export default ({
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
  const [backups, setBackups] = useState<WorldBackup[]>([]);
  const [duplicateName, setDuplicateName] = useState(false);
  const [invalidName, setInvalidName] = useState(false);

  useEffect(() => {
    fetch('/api/backups')
      .then((resp) => resp.json())
      .then((resp) => {
        setBackups(resp);
      });
  }, []);

  const handleChange = (val: string) => {
    setWorldName(val);

    if (!val.match(/^[- _a-zA-Z0-9()]+$/)) {
      setInvalidName(true);
      return;
    }
    if (backups.find((e) => e.worldName === val)) {
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
      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep} disabled={worldName.length === 0 || duplicateName || invalidName}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};
