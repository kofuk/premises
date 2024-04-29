import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';
import {Area, AreaChart, ResponsiveContainer, Tooltip, XAxis, YAxis} from 'recharts';

import {Info as InfoIcon, RestartAlt as RestartIcon, Stop as StopIcon, History as UndoIcon, Public as WorldIcon} from '@mui/icons-material';
import {Box, Button, Card, Stack} from '@mui/material';

import MenuContainer from './menu-container';
import QuickUndo from './quickundo';
import Reconfigure from './reconfigure';
import SystemInfo from './system-info';
import WorldInfo from './world-info';

const ServerControlPane = () => {
  const [t] = useTranslation();

  const [cpuUsage, setCpuUsage] = useState(
    [...Array(100)].map((_) => {
      return {cpuUsage: 0, time: 0};
    })
  );

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/sysstat');
    eventSource.addEventListener('systemstat', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      setCpuUsage((current) => [...current.slice(1, 100), event]);
    });
    return () => {
      eventSource.close();
    };
  }, []);

  return (
    <Card sx={{p: 2, mt: 6}} variant="outlined">
      <MenuContainer
        items={[
          {
            title: t('menu_world_info'),
            icon: <WorldIcon />,
            ui: <WorldInfo />,
            variant: 'dialog',
            cancellable: true
          },
          {
            title: t('menu_reconfigure'),
            icon: <RestartIcon />,
            ui: <Reconfigure />
          },
          {
            title: 'QuickUndo',
            icon: <UndoIcon />,
            ui: <QuickUndo />,
            variant: 'dialog',
            cancellable: true
          },
          {
            title: t('menu_system_info'),
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
                  fetch('/api/stop', {method: 'post'});
                }}
                startIcon={<StopIcon />}
                variant="contained"
              >
                {t('stop_server')}
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
                  formatter={(value: number, _name, _props) => [`${Math.floor(value * 10) / 10}%`, t('cpu_usage')]}
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
