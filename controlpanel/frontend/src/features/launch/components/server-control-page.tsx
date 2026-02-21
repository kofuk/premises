import {Info as InfoIcon, Stop as StopIcon, History as UndoIcon, Public as WorldIcon} from '@mui/icons-material';
import {Box, Button, Card, Stack} from '@mui/material';
import {useTranslation} from 'react-i18next';
import {Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';
import {stop} from '@/api';
import {useAuth} from '@/utils/auth';
import {useRunnerStatus} from '@/utils/runner-status';
import MenuContainer from './menu-container';
import QuickUndo from './quickundo';
import SystemInfo from './system-info';
import WorldInfo from './world-info';

const ServerControlPane = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const {cpuUsage} = useRunnerStatus();

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
            <ResponsiveContainer height={450} width={'100%'}>
              <AreaChart data={cpuUsage} margin={{bottom: 40}}>
                <XAxis
                  angle={-45}
                  dataKey="time"
                  textAnchor="end"
                  tickFormatter={(time) => (time === 0 ? '' : new Date(time).toLocaleTimeString())}
                />
                <YAxis domain={[0, 100]}></YAxis>
                <Tooltip
                  formatter={(value: number | undefined) => [`${Math.floor((value ?? 0) * 10) / 10}%`, t('launch.cpu_usage')]}
                  isAnimationActive={false}
                  labelFormatter={(time) => (time === 0 ? '' : new Date(time).toLocaleTimeString())}
                  wrapperStyle={{opacity: 0.8}}
                />
                <Area dataKey="cpuUsage" dot={false} isAnimationActive={false} stroke="#00f" />
              </AreaChart>
            </ResponsiveContainer>
          </Stack>
        }
      />
    </Card>
  );
};

export default ServerControlPane;
