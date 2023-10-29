import React from 'react';

import {useTranslation} from 'react-i18next';

import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

const WorldSource = ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  worldSource,
  setWorldSource
}: ItemProp & {worldSource: WorldLocation; setWorldSource: (val: WorldLocation) => void}) => {
  const [t] = useTranslation();

  const handleChange = (val: string) => {
    setWorldSource(val === 'backups' ? WorldLocation.Backups : WorldLocation.NewWorld);
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_world_source')}>
      <div className="form-check">
        <input
          checked={worldSource === WorldLocation.Backups}
          className="form-check-input"
          id="worldSourceBackups"
          name="worldSource"
          onChange={(e) => handleChange(e.target.value)}
          type="radio"
          value="backups"
        />
        <label className="form-check-label" htmlFor="worldSourceBackups">
          {t('use_backups')}
        </label>
      </div>
      <div className="form-check">
        <input
          checked={worldSource === WorldLocation.NewWorld}
          className="form-check-input"
          id="worldSourceNewWorld"
          name="worldSource"
          onChange={(e) => handleChange(e.target.value)}
          type="radio"
          value="newWorld"
        />
        <label className="form-check-label" htmlFor="worldSourceNewWorld">
          {t('generate_world')}
        </label>
      </div>

      <div className="m-1 text-end">
        <button className="btn btn-primary" onClick={nextStep} type="button">
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default WorldSource;
