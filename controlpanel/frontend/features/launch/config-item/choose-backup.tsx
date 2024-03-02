import React, {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Alert, Box, Button, FormControl, FormControlLabel, InputLabel, MenuItem, Select, SelectChangeEvent, Switch} from '@mui/material';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

import {APIError, useBackups} from '@/api';
import {Loading} from '@/components';

type Props = ItemProp & {
  worldName: string;
  backupGeneration: string;
  setWorldName: (val: string) => void;
  setBackupGeneration: (val: string) => void;
};

const ChooseBackup = ({isFocused, nextStep, requestFocus, stepNum, worldName, backupGeneration, setWorldName, setBackupGeneration}: Props) => {
  const [t] = useTranslation();

  const {enqueueSnackbar} = useSnackbar();

  const {data: backups, error, isLoading} = useBackups();
  useEffect(() => {
    if (backups && backups.length > 0) {
      if (worldName === '') {
        setWorldName(backups[0].worldName);
      }
    }
  }, [backups]);
  useEffect(() => {
    if (error) {
      if (error instanceof APIError) {
        enqueueSnackbar(error.message, {variant: 'error'});
      }
    }
  }, [error]);

  const changeWorld = (worldName: string) => {
    const generations = backups?.find((e) => e.worldName === worldName)!.generations;
    if (generations) {
      setWorldName(worldName);
    }
  };

  const handleChangeWorld = (event: SelectChangeEvent) => {
    const worldName = event.target.value;
    changeWorld(worldName);
  };

  const handleChangeGeneration = (event: SelectChangeEvent) => {
    setBackupGeneration(event.target.value);
  };

  const createBackupSelector = (): React.ReactElement => {
    const worlds = (
      <FormControl fullWidth>
        <InputLabel id="backup-name-label">{t('select_world')}</InputLabel>
        <Select label={t('select_world')} labelId="backup-name-label" onChange={handleChangeWorld} value={worldName}>
          {backups?.map((e) => (
            <MenuItem key={e.worldName} value={e.worldName}>
              {e.worldName.replace(/^[0-9]+-/, '')}
            </MenuItem>
          ))}
        </Select>
      </FormControl>
    );
    const worldData = backups!.find((e) => e.worldName === worldName);
    const generations = worldData && (
      <FormControl fullWidth>
        <InputLabel id="backup-generation-label">{t('backup_generation')}</InputLabel>
        <Select label={t('backup_generation')} labelId="backup-generation-label" onChange={handleChangeGeneration} value={backupGeneration}>
          {worldData.generations.map((e) => {
            const dateTime = new Date(e.timestamp);
            const label = e.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
              ? dateTime.toLocaleString()
              : `${e.gen} (${dateTime.toLocaleString()})`;
            return (
              <MenuItem key={e.gen} value={e.id}>
                {label}
              </MenuItem>
            );
          })}
        </Select>
      </FormControl>
    );

    return (
      <>
        {worlds}
        <FormControlLabel
          control={
            <Switch
              checked={backupGeneration === '@/latest'}
              onChange={(e) => setBackupGeneration(e.target.checked ? '@/latest' : worldData!.generations[0].id)}
            />
          }
          label={t('use_latest_backup')}
        />
        <Box sx={{my: 3}}>{backupGeneration !== '@/latest' && generations}</Box>
      </>
    );
  };

  const createEmptyMessage = (): React.ReactElement => {
    return (
      <Alert severity="error" sx={{my: 2}}>
        {t('no_backups')}
      </Alert>
    );
  };

  const content = !backups || backups?.length === 0 ? createEmptyMessage() : createBackupSelector();

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_choose_backup')}>
      {isLoading ? (
        <Loading compact />
      ) : (
        <>
          {content}
          <Box sx={{textAlign: 'end'}}>
            <Button endIcon={<NextIcon />} onClick={nextStep} type="button" variant="outlined">
              {t('next')}
            </Button>
          </Box>
        </>
      )}
    </ConfigContainer>
  );
};

export default ChooseBackup;
