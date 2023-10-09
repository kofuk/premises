import React, {useState, useEffect} from 'react';
import {FaStop} from '@react-icons/all-files/fa/FaStop';

import '@/i18n';
import {t} from 'i18next';

import {Line, LineChart, YAxis, Tooltip} from 'recharts';
import ReconfigureMenu from '@/features/launch/reconfigure-menu';
import Snapshot from '@/features/launch/control-item/snapshot';
import QuickUndo from '@/features/launch/control-item/quickundo';
import SystemInfo from '@/features/launch/control-item/system-info';
import WorldInfo from '@/features/launch/control-item/world-info';

enum Modes {
  MainMenu,
  Reconfigure,
  Snapshot,
  QuickUndo,
  SystemInfo,
  WorldInfo
}

type Prop = {
  showError: (message: string) => void;
};

export default (props: Prop) => {
  const {showError} = props;

  const [mode, setMode] = useState(Modes.MainMenu);
  const [cpuUsage, setCpuUsage] = useState(
    Array.apply(null, Array(100)).map((_) => {
      return {cpuUsage: 0};
    })
  );

  useEffect(() => {
    const eventSource = new EventSource('/api/systemstat');
    eventSource.addEventListener('systemstat', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      const data = cpuUsage;
      data.push({cpuUsage: event['cpuUsage']});
      setCpuUsage(data.slice(1));
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
      <div className="list-group" key="mainMenu">
        <button
          type="button"
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.WorldInfo);
          }}
        >
          {t('menu_world_info')}
        </button>
        <button
          type="button"
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.Reconfigure);
          }}
        >
          {t('menu_reconfigure')}
        </button>
        <button
          type="button"
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.Snapshot);
          }}
        >
          {t('menu_snapshot')}
        </button>
        <button
          type="button"
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.QuickUndo);
          }}
        >
          QuickUndo
        </button>
        <button
          type="button"
          className="list-group-item list-group-item-action"
          onClick={() => {
            setMode(Modes.SystemInfo);
          }}
        >
          {t('menu_system_info')}
        </button>
      </div>
    );
  } else if (mode === Modes.Reconfigure) {
    controlItems.push(<ReconfigureMenu backToMenu={handleBackToMenu} showError={showError} key="reconfigure" />);
  } else if (mode === Modes.Snapshot) {
    controlItems.push(<Snapshot backToMenu={handleBackToMenu} showError={showError} key="snapshot" />);
  } else if (mode === Modes.QuickUndo) {
    controlItems.push(<QuickUndo backToMenu={handleBackToMenu} showError={showError} key="quickundo" />);
  } else if (mode === Modes.SystemInfo) {
    controlItems.push(<SystemInfo backToMenu={handleBackToMenu} key="systemInfo" />);
  } else if (mode === Modes.WorldInfo) {
    controlItems.push(<WorldInfo backToMenu={handleBackToMenu} key="worldInfo" />);
  }

  return (
    <div className="my-5 card mx-auto">
      <div className="card-body">
        <form>
          {controlItems}
          <div className="d-md-block mt-3 text-end">
            <button
              className="btn btn-danger bg-gradient"
              type="button"
              onClick={(e: React.MouseEvent) => {
                e.preventDefault();
                fetch('/api/stop', {method: 'post'});
              }}
            >
              <FaStop /> {t('stop_server')}
            </button>
          </div>
        </form>

        <LineChart data={cpuUsage} width={800} height={400}>
          <YAxis domain={[0, 100]}></YAxis>
          <Tooltip />
          <Line dataKey="cpuUsage" stroke="#00f" isAnimationActive={false} dot={false} />
        </LineChart>
      </div>
    </div>
  );
};
