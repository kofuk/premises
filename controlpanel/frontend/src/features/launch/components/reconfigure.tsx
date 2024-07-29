import {useTranslation} from 'react-i18next';

import {PlayArrow as StartIcon} from '@mui/icons-material';
import {Box, Button, Card} from '@mui/material';

import {useLaunchConfig} from './launch-config';
import MenuContainer from './menu-container';
import {create as gameConfigMenu} from './menus/game-config';
import {create as extraGameConfigMenu} from './menus/game-config-extra';
import {create as newWorldSettingsMenu} from './menus/new-world-settings';
import {create as worldMenu} from './menus/world';

const LaunchPage = () => {
  const [t] = useTranslation();

  const {reconfigure, isValid} = useLaunchConfig();

  const handleStart = () => {
    (async () => {
      try {
        await reconfigure();
      } catch (err) {
        console.error(err);
      }
    })();
  };

  return (
    <Card variant="outlined">
      <MenuContainer
        items={[gameConfigMenu(), extraGameConfigMenu(), worldMenu(), newWorldSettingsMenu()]}
        menuFooter={
          <Box sx={{my: 1, textAlign: 'end'}}>
            <Button disabled={!isValid} onClick={handleStart} startIcon={<StartIcon />} sx={{mx: 1}} type="button" variant="contained">
              {t('launch.reconfigure.relaunch')}
            </Button>
          </Box>
        }
      />
    </Card>
  );
};

export default LaunchPage;
