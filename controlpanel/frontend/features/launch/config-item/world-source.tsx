import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Box, Button, FormControlLabel, Radio, RadioGroup} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';

import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

const WorldSource = ({isFocused, nextStep, requestFocus, stepNum}: ItemProp) => {
  const [t] = useTranslation();

  const {updateConfig, config} = useLaunchConfig();

  const [worldSource, setWorldSource] = useState(config.worldSource || WorldLocation.Backups);

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const source = (event.target as HTMLInputElement).value == WorldLocation.Backups ? WorldLocation.Backups : WorldLocation.NewWorld;
    setWorldSource(source);
  };

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({
        worldSource
      });
      nextStep();
    })();
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_world_source')}>
      <RadioGroup onChange={handleChange} value={worldSource}>
        <FormControlLabel control={<Radio />} label={t('use_backups')} value={WorldLocation.Backups} />
        <FormControlLabel control={<Radio />} label={t('generate_world')} value={WorldLocation.NewWorld} />
      </RadioGroup>

      <Box sx={{textAlign: 'end'}}>
        <Button endIcon={<NextIcon />} onClick={saveAndContinue} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default WorldSource;
