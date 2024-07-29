import {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {useRunnerStatus} from '@/utils/runner-status';

const StatusCollector = () => {
  const [t] = useTranslation();

  const {updateStatus} = useRunnerStatus();

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/events');
    eventSource.addEventListener('error', () => {
      updateStatus(0);
    });
    eventSource.addEventListener('statuschanged', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      updateStatus(event.eventCode, event.extra, event.pageCode);
    });

    return () => {
      eventSource.close();
    };
  }, []);

  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/info');
    eventSource.addEventListener('error', () => {
      enqueueSnackbar(t('navbar.reconnecting'), {variant: 'error'});
    });
    eventSource.addEventListener('notify', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      const variant = event.isError ? 'error' : 'success';
      enqueueSnackbar(t(`info.code_${event.infoCode}`), {variant});
    });

    return () => {
      eventSource.close();
    };
  }, []);

  return <></>;
};

export default StatusCollector;
