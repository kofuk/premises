import React, {ChangeEvent, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {
  Alert,
  Box,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem as MUIMenuItem,
  Radio,
  RadioGroup,
  Select,
  SelectChangeEvent,
  Switch,
  TextField
} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {valueLabel} from './common';

import {useBackups} from '@/api';
import {Loading} from '@/components';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

const SavedWorld = ({name, setName, gen, setGen}: {name: string; setName: (name: string) => void; gen: string; setGen: (gen: string) => void}) => {
  const [t] = useTranslation();

  const {data: backups, isLoading} = useBackups();
  if (isLoading) {
    return <Loading compact />;
  }

  if (!backups || backups.length === 0) {
    return <Alert severity="error">{t('no_backups')}</Alert>;
  }

  const handleChangeWorld = (event: SelectChangeEvent) => {
    const name = event.target.value;
    setName(name);
    setGen('@/latest');
  };

  const handleChangeGen = (event: SelectChangeEvent) => {
    setGen(event.target.value);
  };

  const selectedWorld = backups!.find((e) => e.worldName === name);

  return (
    <FormControl fullWidth>
      <InputLabel id="backup-name-label">{t('select_world')}</InputLabel>
      <Select label={t('select_world')} labelId="backup-name-label" onChange={handleChangeWorld} value={name}>
        {backups?.map((e) => (
          <MUIMenuItem key={e.worldName} value={e.worldName}>
            {e.worldName.replace(/^[0-9]+-/, '')}
          </MUIMenuItem>
        ))}
      </Select>
      <FormControlLabel
        control={
          <Switch
            checked={gen === '@/latest'}
            onChange={(e) => setGen(!selectedWorld || e.target.checked ? '@/latest' : selectedWorld!.generations[0].id)}
          />
        }
        label={t('use_latest_backup')}
      />
      {selectedWorld && gen !== '@/latest' && (
        <Select label={t('backup_generation')} labelId="backup-generation-label" onChange={handleChangeGen} value={gen}>
          {selectedWorld.generations.map((e) => {
            const dateTime = new Date(e.timestamp);
            const label = e.gen.match(/[0-9]+-[0-9]+-[0-9]+ [0-9]+:[0-9]+:[0-9]+/)
              ? dateTime.toLocaleString()
              : `${e.gen} (${dateTime.toLocaleString()})`;
            return (
              <MUIMenuItem key={e.gen} value={e.id}>
                {label}
              </MUIMenuItem>
            );
          })}
        </Select>
      )}
    </FormControl>
  );
};

const NewWorld = ({name, setName}: {name: string; setName: (name: string) => void}) => {
  const [t] = useTranslation();

  const handleChange = (e: ChangeEvent<HTMLInputElement>) => {
    setName(e.target.value);
  };

  return <TextField fullWidth inputProps={{'data-1p-ignore': ''}} label={t('world_name')} onChange={handleChange} type="text" value={name} />;
};

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const [worldSource, setWorldSource] = useState(config.worldSource || WorldLocation.Backups);
  const [name, setName] = useState(config.worldName || '');
  const [gen, setGen] = useState(config.backupGen || '@/latest');

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const source = (event.target as HTMLInputElement).value == WorldLocation.Backups ? WorldLocation.Backups : WorldLocation.NewWorld;
    setWorldSource(source);
    setName('');
    setGen('@/latest');
  };

  const notSetLabel = valueLabel(null);

  const createLabel = () => {
    if (!config.worldName) {
      return notSetLabel;
    }

    if (config.worldSource === WorldLocation.Backups) {
      return `${t('use_backups')} (${config.worldName})`;
    } else {
      return `${t('generate_world')} (${config.worldName})`;
    }
  };

  return {
    title: t('config_world_source'),
    ui: (
      <Box>
        <RadioGroup onChange={handleChange} value={worldSource}>
          <FormControlLabel control={<Radio />} label={t('use_backups')} value={WorldLocation.Backups} />
          <FormControlLabel control={<Radio />} label={t('generate_world')} value={WorldLocation.NewWorld} />
        </RadioGroup>
        <Box sx={{mt: 2}}>
          {worldSource === WorldLocation.Backups ? (
            <SavedWorld gen={gen} name={name} setGen={setGen} setName={setName} />
          ) : (
            <NewWorld name={name} setName={setName} />
          )}
        </Box>
      </Box>
    ),
    detail: createLabel(),
    variant: 'dialog',
    cancellable: true,
    action: {
      label: t('save'),
      callback: () => {
        updateConfig({worldSource, worldName: name, backupGen: gen});
      }
    }
  };
};
