import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {FormControl, InputLabel, MenuItem as MUIMenuItem, Select, Stack, TextField} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {WorldLocation} from './world';

export enum LevelType {
  Default = 'default',
  Superflat = 'flat',
  LargeBiomes = 'largeBiomes',
  Amplified = 'amplified'
}

type LevelTypeInfo = {
  levelType: LevelType;
  label: string;
};

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const [levelType, setLevelType] = useState(config.levelType || LevelType.Default);
  const [seed, setSeed] = useState(config.seed || '');

  const levelTypes: LevelTypeInfo[] = [
    {levelType: LevelType.Default, label: t('world_type_default')},
    {levelType: LevelType.Superflat, label: t('world_type_superflat')},
    {levelType: LevelType.LargeBiomes, label: t('world_type_large_biomes')},
    {levelType: LevelType.Amplified, label: t('world_type_amplified')}
  ];

  const label = config.seed ? t('world_settings_label', {seed: config.seed, levelType: config.levelType}) : t('default_settings');

  return {
    title: t('config_configure_world'),
    ui: (
      <Stack spacing={3} sx={{mt: 1}}>
        <TextField
          fullWidth
          inputProps={{'data-1p-ignore': ''}}
          label={t('seed')}
          onChange={(e) => {
            setSeed(e.target.value);
          }}
          type="text"
          value={seed}
        />

        <FormControl fullWidth>
          <InputLabel id="level-type-label">{t('world_type')}</InputLabel>
          <Select label={t('world_type')} labelId="level-type-label" onChange={(e) => setLevelType(e.target.value as LevelType)} value={levelType}>
            {levelTypes.map((e) => (
              <MUIMenuItem key={e.levelType} value={e.levelType}>
                {e.label}
              </MUIMenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>
    ),
    detail: label,
    variant: 'dialog',
    cancellable: true,
    disabled: config.worldSource !== WorldLocation.NewWorld,
    action: {
      label: t('save'),
      callback: () => {
        updateConfig({seed, levelType});
      }
    }
  };
};
