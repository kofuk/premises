import {Info as InfoIcon, Stop as StopIcon, History as UndoIcon, Public as WorldIcon} from '@mui/icons-material';
import {Box, Button, Card, Stack} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {stop} from '@/api';
import {useAuth} from '@/utils/auth';
import MenuContainer from './menu-container';
import QuickUndo from './quickundo';
import SystemInfo from './system-info';
import WorldInfo from './world-info';

const ServerControlPane = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  return (
    <Card sx={{p: 2, mt: 6}} variant="outlined">
      <MenuContainer
        items={[
          {
            title: t('launch.world_info'),
            icon: <WorldIcon />,
            ui: <WorldInfo />,
            variant: 'dialog',
            cancellable: true
          },
          {
            title: t('launch.quick_undo'),
            icon: <UndoIcon />,
            ui: <QuickUndo />,
            variant: 'dialog',
            cancellable: true
          },
          {
            title: t('launch.system_info'),
            icon: <InfoIcon />,
            ui: <SystemInfo />,
            variant: 'dialog',
            cancellable: true
          }
        ]}
        menuFooter={
          <Stack spacing={1}>
            <Box sx={{textAlign: 'end'}}>
              <Button
                onClick={() => {
                  stop(accessToken);
                }}
                startIcon={<StopIcon />}
                variant="contained"
              >
                {t('launch.stop')}
              </Button>
            </Box>
          </Stack>
        }
      />
    </Card>
  );
};

export default ServerControlPane;
