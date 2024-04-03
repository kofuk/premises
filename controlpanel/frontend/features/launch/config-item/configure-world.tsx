import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Box, Button, FormControl, InputLabel, MenuItem, Select, Stack, TextField} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

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

const ConfigureWorld = ({isFocused, nextStep, requestFocus, stepNum}: ItemProp) => {
  const [t] = useTranslation();

  const {updateConfig, config} = useLaunchConfig();

  const [levelType, setLevelType] = useState(config.levelType || LevelType.Default);
  const [seed, setSeed] = useState(config.seed || '');

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({
        levelType,
        seed
      });
      nextStep();
    })();
  };

  const levelTypes: LevelTypeInfo[] = [
    {levelType: LevelType.Default, label: t('world_type_default')},
    {levelType: LevelType.Superflat, label: t('world_type_superflat')},
    {levelType: LevelType.LargeBiomes, label: t('world_type_large_biomes')},
    {levelType: LevelType.Amplified, label: t('world_type_amplified')}
  ];

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_configure_world')}>
      <Stack spacing={3}>
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
              <MenuItem key={e.levelType} value={e.levelType}>
                {e.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>

      <Box sx={{textAlign: 'end', mt: 1}}>
        <Button endIcon={<NextIcon />} onClick={saveAndContinue} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default ConfigureWorld;
