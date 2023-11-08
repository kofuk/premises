import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';
import {Line, LineChart, Tooltip, YAxis} from 'recharts';

import {Stop as StopIcon} from '@mui/icons-material';

import QuickUndo from '@/features/launch/control-item/quickundo';
import SystemInfo from '@/features/launch/control-item/system-info';
import WorldInfo from '@/features/launch/control-item/world-info';
import ReconfigureMenu from '@/features/launch/reconfigure-menu';

enum Modes {
  MainMenu,
  Reconfigure,
  QuickUndo,
  SystemInfo,
  WorldInfo
}

const ServerControlPane = () => {
  const [t] = useTranslation();

  const [mode, setMode] = useState(Modes.MainMenu);
  const [cpuUsage, setCpuUsage] = useState(
    [...Array(100)].map((_) => {
      return {cpuUsage: 0};
    })
  );

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/sysstat');
    eventSource.addEventListener('systemstat', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      setCpuUsage((current) => [...current.slice(1, 101), event]);
    });
    return () => {
      eventSource.close();
    };
  }, []);

  const handleBackToMenu = () => {
    setMode(Modes.MainMenu);
  };

  const controlItems: React.ReactElement[] = [];

  if (mode === Modes.MainMenu) {
    controlItems.push(
      <div key="mainMenu" className="list-group">
        <button
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.WorldInfo);
          }}
          type="button"
        >
          {t('menu_world_info')}
        </button>
        <button
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.Reconfigure);
          }}
          type="button"
        >
          {t('menu_reconfigure')}
        </button>
        <button
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.QuickUndo);
          }}
          type="button"
        >
          QuickUndo
        </button>
        <button
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.SystemInfo);
          }}
          type="button"
        >
          {t('menu_system_info')}
        </button>
      </div>
    );
  } else if (mode === Modes.Reconfigure) {
    controlItems.push(<ReconfigureMenu key="reconfigure" backToMenu={handleBackToMenu} />);
  } else if (mode === Modes.QuickUndo) {
    controlItems.push(<QuickUndo key="quickundo" backToMenu={handleBackToMenu} />);
  } else if (mode === Modes.SystemInfo) {
    controlItems.push(<SystemInfo key="systemInfo" backToMenu={handleBackToMenu} />);
  } else if (mode === Modes.WorldInfo) {
    controlItems.push(<WorldInfo key="worldInfo" backToMenu={handleBackToMenu} />);
  }

  return (
    <div className="my-5 card mx-auto">
      <div className="card-body">
        <form>
          {controlItems}
          <div className="d-md-block mt-3 text-end">
            <button
              className="btn btn-danger bg-gradient"
              onClick={(e: React.MouseEvent) => {
                e.preventDefault();
                fetch('/api/stop', {method: 'post'});
              }}
              type="button"
            >
              <StopIcon /> {t('stop_server')}
            </button>
          </div>
        </form>

        <LineChart data={cpuUsage} height={400} width={800}>
          <YAxis domain={[0, 100]}></YAxis>
          <Tooltip />
          <Line dataKey="cpuUsage" dot={false} isAnimationActive={false} stroke="#00f" />
        </LineChart>
      </div>
    </div>
  );
};

export default ServerControlPane;
