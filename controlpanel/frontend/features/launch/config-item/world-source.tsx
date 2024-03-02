import React from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Box, Button, FormControlLabel, Radio, RadioGroup} from '@mui/material';

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

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setWorldSource((event.target as HTMLInputElement).value == WorldLocation.Backups ? WorldLocation.Backups : WorldLocation.NewWorld);
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_world_source')}>
      <RadioGroup onChange={handleChange} value={worldSource}>
        <FormControlLabel control={<Radio />} label={t('use_backups')} value={WorldLocation.Backups} />
        <FormControlLabel control={<Radio />} label={t('generate_world')} value={WorldLocation.NewWorld} />
      </RadioGroup>

      <Box sx={{textAlign: 'end'}}>
        <Button endIcon={<NextIcon />} onClick={nextStep} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default WorldSource;
