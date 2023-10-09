import '@/i18n';
import {t} from 'i18next';

import {ItemProp} from '@/features/launch/config-item/prop';
import ConfigContainer from '@/features/launch/config-item/config-container';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

export default ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  worldSource,
  setWorldSource
}: ItemProp & {worldSource: WorldLocation; setWorldSource: (val: WorldLocation) => void}) => {
  const handleChange = (val: string) => {
    setWorldSource(val === 'backups' ? WorldLocation.Backups : WorldLocation.NewWorld);
  };

  return (
    <ConfigContainer title={t('config_world_source')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      <div className="form-check">
        <input
          className="form-check-input"
          type="radio"
          name="worldSource"
          value="backups"
          id="worldSourceBackups"
          checked={worldSource === WorldLocation.Backups}
          onChange={(e) => handleChange(e.target.value)}
        />
        <label className="form-check-label" htmlFor="worldSourceBackups">
          {t('use_backups')}
        </label>
      </div>
      <div className="form-check">
        <input
          className="form-check-input"
          type="radio"
          name="worldSource"
          value="newWorld"
          id="worldSourceNewWorld"
          checked={worldSource === WorldLocation.NewWorld}
          onChange={(e) => handleChange(e.target.value)}
        />
        <label className="form-check-label" htmlFor="worldSourceNewWorld">
          {t('generate_world')}
        </label>
      </div>

      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};
