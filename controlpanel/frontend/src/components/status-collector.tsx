import {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {useAuth} from '@/utils/auth';
import {useRunnerStatus} from '@/utils/runner-status';

const StatusCollector = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const {updateStatus, updateCpuUsage} = useRunnerStatus();
  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    const params = new URLSearchParams();
    params.set('x-auth', `Bearer ${accessToken}`);

    const eventSource = new EventSource(`/api/streaming?${params.toString()}`);
    eventSource.addEventListener('error', () => {
      updateStatus(0);
    });
    eventSource.addEventListener('event', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      updateStatus(event.eventCode, event.extra, event.pageCode);
    });
    eventSource.addEventListener('notify', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      const variant = event.isError ? 'error' : 'success';
      enqueueSnackbar(t(`info.code_${event.infoCode}`), {variant});
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
