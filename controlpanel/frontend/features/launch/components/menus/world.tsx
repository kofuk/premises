import React from 'react';

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
  Stack,
  Switch
} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {valueLabel} from './common';

import {useWorlds} from '@/api';
import Loading from '@/components/loading';
import SaveInput from '@/components/save-input';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

const SavedWorld = ({name, setName, gen, setGen}: {name: string; setName: (name: string) => void; gen: string; setGen: (gen: string) => void}) => {
  const [t] = useTranslation();

  const {data: savedWorlds, isLoading} = useWorlds();
  if (isLoading) {
    return <Loading compact />;
  }

  if (!savedWorlds || savedWorlds.length === 0) {
    return <Alert severity="error">{t('launch.world.no_world')}</Alert>;
  }

  const handleChangeWorld = (event: SelectChangeEvent) => {
    const name = event.target.value;
    setName(name);
    setGen('@/latest');
  };

  const handleChangeGen: (event: SelectChangeEvent) => void = (event: SelectChangeEvent) => {
    setGen(event.target.value);
  };

  const selectedWorld = savedWorlds!.find((e) => e.worldName === name);

  return (
    <Stack spacing={1}>
      <FormControl fullWidth>
        <InputLabel>{t('launch.world.select')}</InputLabel>
        <Select label={t('launch.world.select')} onChange={handleChangeWorld} value={name}>
          {savedWorlds?.map((e) => (
            <MUIMenuItem key={e.worldName} value={e.worldName}>
              {e.worldName.replace(/^[0-9]+-/, '')}
            </MUIMenuItem>
          ))}
        </Select>
      </FormControl>
      <FormControlLabel
        control={
          <Switch
            checked={gen === '@/latest'}
            onChange={(e) => setGen(!selectedWorld || e.target.checked ? '@/latest' : selectedWorld!.generations[0].id)}
          />
        }
        label={t('launch.world.use_latest_world')}
      />
      {selectedWorld && gen !== '@/latest' && (
        <FormControl fullWidth>
          <InputLabel>{t('launch.world.world_generation')}</InputLabel>
          <Select label={t('launch.world.world_generation')} onChange={handleChangeGen} value={gen}>
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
        </FormControl>
      )}
    </Stack>
  );
};

const NewWorld = ({name, setName}: {name: string; setName: (name: string) => void}) => {
  const [t] = useTranslation();
  return <SaveInput fullWidth initValue={name} label={t('launch.world.name')} onSave={setName} type="text" unsuitableForPasswordAutoFill />;
};

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const worldSource = config.worldSource || WorldLocation.Backups;
  const name = config.worldName || '';
  const gen = config.backupGen || '@/latest';

  const handleChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const source = (event.target as HTMLInputElement).value === WorldLocation.Backups ? WorldLocation.Backups : WorldLocation.NewWorld;

    updateConfig({worldSource: source, worldName: '', backupGen: '@/latest'});
  };

  const setName = (name: string) => {
    updateConfig({worldName: name});
  };

  const setGen = (gen: string) => {
    updateConfig({backupGen: gen});
  };

  const notSetLabel = valueLabel(null);

  const createLabel = () => {
    if (!config.worldName) {
      return notSetLabel;
    }

    if (config.worldSource === WorldLocation.Backups) {
      return t('launch.world.summary_existing', {name: config.worldName});
    } else {
      return t('launch.world.summary_new', {name: config.worldName});
    }
  };

  return {
    title: t('launch.world'),
    ui: (
      <Box>
        <RadioGroup onChange={handleChange} value={worldSource}>
          <FormControlLabel control={<Radio />} label={t('launch.world.load_existing')} value={WorldLocation.Backups} />
          <FormControlLabel control={<Radio />} label={t('launch.world.create_new')} value={WorldLocation.NewWorld} />
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
    cancellable: true
  };
};
