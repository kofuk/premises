import React from 'react';

import {useTranslation} from 'react-i18next';

import {PlayArrow as StartIcon} from '@mui/icons-material';
import {Box, Button, Card} from '@mui/material';

import {useLaunchConfig} from './launch-config';
import MenuContainer from './menu-container';
import {create as gameConfigMenu} from './menus/game-config';
import {create as extraGameConfigMenu} from './menus/game-config-extra';
import {create as machineTypeMenu} from './menus/machine-type';
import {create as newWorldSettingsMenu} from './menus/new-world-settings';
import {create as worldMenu} from './menus/world';

const LaunchPage = () => {
  const [t] = useTranslation();

  const {launch, isValid} = useLaunchConfig();

  const handleStart = () => {
    (async () => {
      try {
        await launch();
      } catch (err) {
        console.error(err);
      }
    })();
  };

  return (
    <Card sx={{p: 2, mt: 6}} variant="outlined">
      <MenuContainer
        items={[machineTypeMenu(), gameConfigMenu(), extraGameConfigMenu(), worldMenu(), newWorldSettingsMenu()]}
        menuFooter={
          <Box sx={{textAlign: 'end'}}>
            <Button disabled={!isValid} onClick={handleStart} startIcon={<StartIcon />} sx={{mx: 1}} type="button" variant="contained">
              {t('launch.launch')}
            </Button>
          </Box>
        }
      />
    </Card>
  );
};

export default LaunchPage;
