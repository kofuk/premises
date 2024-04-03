import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Alert, Box, Button, TextField} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';

import {useBackups} from '@/api';
import {Loading} from '@/components';
import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

const WorldName = ({isFocused, nextStep, requestFocus, stepNum}: ItemProp) => {
  const [t] = useTranslation();

  const {updateConfig, config} = useLaunchConfig();

  const [worldName, setWorldName] = useState(config.worldName || '');

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({
        worldName
      });
      nextStep();
    })();
  };

  const {data: backups, isLoading} = useBackups();

  const handleChange = (val: string) => {
    setWorldName(val);
  };

  const invalidName = !worldName.match(/^[- _a-zA-Z0-9()]+$/);
  const duplicateName = !!backups?.find((e) => e.worldName === worldName);

  const createAlert = () => {
    if (invalidName) {
      return (
        <Alert severity="error" sx={{my: 2}}>
          {t('world_name_error_invalid')}
        </Alert>
      );
    }
    if (duplicateName) {
      return (
        <Alert severity="error" sx={{my: 2}}>
          {t('world_name_error_duplicate')}
        </Alert>
      );
    }

    return null;
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_world_name')}>
      <Box sx={{my: 2}}>
        {(isLoading && <Loading compact />) || (
          <>
            <TextField
              fullWidth
              inputProps={{'data-1p-ignore': ''}}
              label={t('world_name')}
              onChange={(e) => {
                handleChange(e.target.value);
              }}
              type="text"
              value={worldName}
            />
            {createAlert()}
          </>
        )}
      </Box>
      <Box sx={{textAlign: 'end', mt: 1}}>
        <Button
          disabled={worldName.length === 0 || duplicateName || invalidName}
          endIcon={<NextIcon />}
          onClick={saveAndContinue}
          type="button"
          variant="outlined"
        >
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default WorldName;
