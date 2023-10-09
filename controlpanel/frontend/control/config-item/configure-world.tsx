import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import ConfigContainer from './config-container';

export enum LevelType {
  Default = 'default',
  Superflat = 'flat',
  LargeBiomes = 'largeBiomes',
  Amplified = 'amplified'
}

class LevelTypeInfo {
  levelType: LevelType;
  label: string;

  constructor(levelType: LevelType, label: string) {
    this.levelType = levelType;
    this.label = label;
  }

  createReactElement = (): React.ReactElement => {
    return <option value={this.levelType}>{this.label}</option>;
  };
}

const levelTypes: LevelTypeInfo[] = [
  new LevelTypeInfo(LevelType.Default, t('world_type_default')),
  new LevelTypeInfo(LevelType.Superflat, t('world_type_superflat')),
  new LevelTypeInfo(LevelType.LargeBiomes, t('world_type_large_biomes')),
  new LevelTypeInfo(LevelType.Amplified, t('world_type_amplified'))
];

export default ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  levelType,
  seed,
  setLevelType,
  setSeed
}: ItemProp & {
  levelType: LevelType;
  seed: string;
  setLevelType: (val: LevelType) => void;
  setSeed: (val: string) => void;
}) => {
  return (
    <ConfigContainer title={t('config_configure_world')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      <div className="m-2">
        <label className="form-label" htmlFor="seed">
          {t('seed')}
        </label>
        <input className="form-control" value={seed} id="seed" onChange={(e) => setSeed(e.target.value)} />
      </div>

      <div className="m-2">
        <label className="form-label" htmlFor="selectLevelType">
          {t('world_type')}
        </label>
        <select className="form-select" value={levelType} id="selectLeveltype" onChange={(e) => setLevelType(e.target.value as LevelType)}>
          {levelTypes.map((e) => e.createReactElement())}
        </select>
      </div>

      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};
