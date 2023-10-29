import React from 'react';

import {useTranslation} from 'react-i18next';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

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
    return (
      <option key={this.levelType} value={this.levelType}>
        {this.label}
      </option>
    );
  };
}

const ConfigureWorld = ({
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
  const [t] = useTranslation();

  const levelTypes: LevelTypeInfo[] = [
    new LevelTypeInfo(LevelType.Default, t('world_type_default')),
    new LevelTypeInfo(LevelType.Superflat, t('world_type_superflat')),
    new LevelTypeInfo(LevelType.LargeBiomes, t('world_type_large_biomes')),
    new LevelTypeInfo(LevelType.Amplified, t('world_type_amplified'))
  ];

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_configure_world')}>
      <div className="m-2">
        <label className="form-label" htmlFor="seed">
          {t('seed')}
        </label>
        <input className="form-control" id="seed" onChange={(e) => setSeed(e.target.value)} value={seed} />
      </div>

      <div className="m-2">
        <label className="form-label" htmlFor="selectLevelType">
          {t('world_type')}
        </label>
        <select className="form-select" id="selectLeveltype" onChange={(e) => setLevelType(e.target.value as LevelType)} value={levelType}>
          {levelTypes.map((e) => e.createReactElement())}
        </select>
      </div>

      <div className="m-1 text-end">
        <button className="btn btn-primary" onClick={nextStep} type="button">
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default ConfigureWorld;
