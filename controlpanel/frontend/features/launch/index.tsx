import React, {useEffect, useState} from 'react';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import StatusBar from './components/statusbar';
import ServerConfigPane from './server-config-pane';
import ServerControlPane from './server-control-pane';

// For bootstrap based screen. We should remove this after migrating to styled-component completed.
import 'bootstrap/scss/bootstrap.scss';
/////

const PAGE_LAUNCH = 1;

const LaunchPage = () => {
  const [t] = useTranslation();

  const [useNotification, setUseNotification] = useState(false);
  const [message, setMessage] = useState(t('connecting'));
  const [progress, setProgress] = useState(0);
  const [prevStatus, setPrevStatus] = useState('');
  const [page, setPage] = useState(PAGE_LAUNCH);

  useEffect(() => {
    const eventSource = new EventSource('/api/status');
    eventSource.addEventListener('error', () => {
      setMessage(t('reconnecting'));
    });
    eventSource.addEventListener('statuschanged', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      setMessage(t(`status.code_${event.eventCode}`));
      setProgress(event.progress);
      setPage(event.pageCode);

      //TODO: temporary implementation
      if (useNotification) {
        if (event.message !== prevStatus && prevStatus !== '' && (event.message === '実行中' || event.message === 'Running')) {
          new Notification(t('notification_title'), {body: t('notification_body')});
        }
      }

      setPrevStatus(event.status);
    });

    return () => {
      eventSource.close();
    };
  }, []);

  const [showNotificationToast, setShowNotificationToast] = useState(true);
  useEffect(() => {
    if (Notification.permission === 'granted') {
      setShowNotificationToast(false);
      setUseNotification(true);
    }
  }, []);

  const closeNotificationToast = () => {
    setShowNotificationToast(false);
  };

  const requestNotification = () => {
    (async () => {
      try {
        const result = await Notification.requestPermission();
        const granted = result === 'granted';
        setUseNotification(granted);
        if (granted) {
          closeNotificationToast();
        }
      } catch (err) {
        console.error(err);
      }
    })();
  };

  const mainPane: React.ReactElement = page == PAGE_LAUNCH ? <ServerConfigPane /> : <ServerControlPane />;
  return (
    <>
      <StatusBar message={message} progress={progress} />
      {mainPane}

      <div className="toast-container position-absolute top-0 end-0 pe-1 pt-5 mt-3">
        <div className={`toast ${showNotificationToast ? 'show' : ''}`}>
          <div className="toast-header">
            <strong className="me-auto">{t('notification_toast_title')}</strong>
            <button aria-label="Close" className="btn-close" data-bs-dismiss="toast" onClick={closeNotificationToast} type="button"></button>
          </div>
          <div className="toast-body">
            {t('notification_toast_description')}
            <div className="text-end">
              <button className="btn btn-primary btn-sm" onClick={requestNotification} type="button">
                {t('notification_allow')}
              </button>
            </div>
          </div>
        </div>
      </div>
      <Helmet>
        <title>{t('app_name')}</title>
      </Helmet>
    </>
  );
};

export default LaunchPage;
