import {useEffect} from 'react';

import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

import {useAuth} from '@/utils/auth';
import {useRunnerStatus} from '@/utils/runner-status';

const StatusCollector = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const {updateStatus, updateCpuUsage} = useRunnerStatus();

  useEffect(() => {
    const params = new URLSearchParams();
    params.set('x-auth', `Bearer ${accessToken}`);

    const eventSource = new EventSource(`/api/v1/streaming?${params.toString()}`);
    eventSource.addEventListener('error', () => {
      updateStatus(0);
    });
    eventSource.addEventListener('event', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      updateStatus(event.eventCode, event.extra, event.pageCode);
    });
    eventSource.addEventListener('notify', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      if (event.isError) {
        toast.error(t(`info.code_${event.infoCode}`));
      } else {
        toast.info(t(`info.code_${event.infoCode}`));
      }
    });
    eventSource.addEventListener('sysstat', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      updateCpuUsage(event);
    });

    return () => {
      eventSource.close();
    };
  }, []);

  return <></>;
};

export default StatusCollector;
