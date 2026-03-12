import {Box, FormControlLabel, Radio, RadioGroup, Stack} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {useWorlds} from '@/api';
import Loading from '@/components/loading';
import SaveInput from '@/components/save-input';
import {useAuth} from '@/utils/auth';
import {useLaunchConfig} from '../launch-config';
import type {MenuItem} from '../menu-container';
import WorldExplorer from '../world-explorer';
import {valueLabel} from './common';

export enum WorldLocation {
  Backups = 'backups',
  NewWorld = 'new-world'
}

const SavedWorld = ({name, setName, gen, setGen}: {name: string; setName: (name: string) => void; gen: string; setGen: (gen: string) => void}) => {
  const {accessToken} = useAuth();

  const {data: savedWorlds, isLoading, mutate} = useWorlds(accessToken);
  if (isLoading) {
    return <Loading compact />;
  }

  return (
    <Stack spacing={1}>
      {savedWorlds && (
        <WorldExplorer
          onChange={(selection) => {
            setName(selection.worldName);
            setGen(selection.generationId);
          }}
          refresh={() => mutate()}
          selection={{worldName: name, generationId: gen}}
          worlds={savedWorlds}
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
      if (config.backupGen === '@/latest') {
        return t('launch.world.summary_existing_latest', {name: config.worldName});
      }
      return t('launch.world.summary_existing', {name: config.worldName});
    } else {
      return t('launch.world.summary_new', {name: config.worldName});
    }
  };

  return {
    title: t('launch.world'),
    ui: (
      <Box>
        <RadioGroup onChange={handleChange} row value={worldSource}>
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
