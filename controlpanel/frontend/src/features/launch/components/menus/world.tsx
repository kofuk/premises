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
import WorldExplorer from '../world-explorer';

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

  if (!savedWorlds) {
    return <Alert severity="error">{t('launch.world.no_world')}</Alert>;
  }

  return (
    <Stack spacing={1}>
      {savedWorlds && (
        <WorldExplorer
          worlds={savedWorlds}
          selection={{worldName: name, generationId: gen}}
          onChange={(selection) => {
            setName(selection.worldName);
            setGen(selection.generationId);
          }}
        />
      )}
    </Stack>
  );
};

const NewWorld = ({name, setName}: {name: string; setName: (name: string) => void}) => {
  const [t] = useTranslation();
  const handleSaveName = (name: string) => {
    setName(name.replaceAll(/[-@]/g, '-'));
  };
  return <SaveInput fullWidth initValue={name} label={t('launch.world.name')} onSave={handleSaveName} type="text" unsuitableForPasswordAutoFill />;
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
