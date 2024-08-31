import {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {useRunnerStatus} from '@/utils/runner-status';

const StatusCollector = () => {
  const [t] = useTranslation();

  const {updateStatus, updateCpuUsage} = useRunnerStatus();
  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming');
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
