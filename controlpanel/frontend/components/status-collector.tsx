import React, {useEffect} from 'react';

import {useSnackbar} from 'notistack';
import {useTranslation} from 'react-i18next';

import {useRunnerStatus} from '@/utils/runner-status';

const StatusCollector = () => {
  const [t] = useTranslation();

  const {updateStatus} = useRunnerStatus();

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/events');
    eventSource.addEventListener('error', () => {
      updateStatus(0, 0);
    });
    eventSource.addEventListener('statuschanged', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      updateStatus(event.eventCode, event.progress, event.pageCode);
    });

    return () => {
      eventSource.close();
    };
  }, []);

  const {enqueueSnackbar} = useSnackbar();

  useEffect(() => {
    const eventSource = new EventSource('/api/streaming/error');
    eventSource.addEventListener('error', () => {
      enqueueSnackbar(t('reconnecting'), {variant: 'error'});
    });
    eventSource.addEventListener('trigger', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      enqueueSnackbar(t(`error.code_${event.errorCode}`), {variant: 'error'});
    });

    return () => {
      eventSource.close();
    };
  }, []);

  return <></>;
};

export default StatusCollector;
