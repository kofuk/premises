import {useTranslation} from 'react-i18next';

import {FormControl, InputLabel, MenuItem as MUIMenuItem, Select, Stack} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {WorldLocation} from './world';

import SaveInput from '@/components/save-input';

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

  const levelType = config.levelType || LevelType.Default;
  const seed = config.seed || '';

  const setLevelType = (levelType: string) => {
    updateConfig({levelType});
  };

  const setSeed = (seed: string) => {
    updateConfig({seed, levelType});
  };

  const levelTypes: LevelTypeInfo[] = [
    {levelType: LevelType.Default, label: t('launch.new_world.level_default')},
    {levelType: LevelType.Superflat, label: t('launch.new_world.level_superflat')},
    {levelType: LevelType.LargeBiomes, label: t('launch.new_world.level_large_biomes')},
    {levelType: LevelType.Amplified, label: t('launch.new_world.level_amplified')}
  ];

  const levelTypeName = levelTypes.find((e) => e.levelType === (config.levelType || 'default'))?.label;

  const detail =
    seed === ''
      ? t('launch.new_world.summary', {levelType: levelTypeName})
      : t('launch.new_world.summary_with_seed', {levelType: levelTypeName, seed: seed});

  return {
    title: t('launch.new_world'),
    ui: (
      <Stack spacing={3} sx={{mt: 1}}>
        <SaveInput fullWidth initValue={seed} label={t('launch.new_world.seed')} onSave={setSeed} type="text" />

        <FormControl fullWidth>
          <InputLabel id="level-type-label">{t('launch.new_world.level_type')}</InputLabel>
          <Select
            label={t('launch.new_world.level_type')}
            labelId="level-type-label"
            onChange={(e) => setLevelType(e.target.value as LevelType)}
            value={levelType}
          >
            {levelTypes.map((e) => (
              <MUIMenuItem key={e.levelType} value={e.levelType}>
                {e.label}
              </MUIMenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>
    ),
    detail,
    variant: 'dialog',
    cancellable: true,
    disabled: config.worldSource !== WorldLocation.NewWorld
  };
};
