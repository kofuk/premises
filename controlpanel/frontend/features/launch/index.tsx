import React, {useState, useEffect} from 'react';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import ServerConfigPane from './server-config-pane';
import ServerControlPane from './server-control-pane';
import StatusBar from './statusbar';

const LaunchPage = () => {
  const [t] = useTranslation();

  const [useNotification, setUseNotification] = useState(false);
  const [isServerShutdown, setIsServerShutdown] = useState(true);
  const [isError, setIsError] = useState(false);
  const [message, setMessage] = useState(t('connecting'));
  const [prevStatus, setPrevStatus] = useState('');

  useEffect(() => {
    const eventSource = new EventSource('/api/status');
    eventSource.addEventListener('error', () => {
      setIsError(true);
      setMessage(t('reconnecting'));
    });
    eventSource.addEventListener('statuschanged', (ev: MessageEvent) => {
      const event = JSON.parse(ev.data);
      setIsServerShutdown(event.shutdown);
      setIsError(event.hasError);
      setMessage(event.status);

      //TODO: temporary implementation
      if (useNotification) {
        if (event.status !== prevStatus && prevStatus !== '' && (event.status === '実行中' || event.status === 'Running')) {
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

  const showError = (message: string) => {
    setIsError(true);
    setMessage(message);
  };

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

  const mainPane: React.ReactElement = isServerShutdown ? <ServerConfigPane showError={showError} /> : <ServerControlPane showError={showError} />;
  return (
    <>
      <StatusBar isError={isError} message={message} />
      {mainPane}

      <div className="toast-container position-absolute top-0 end-0 pe-1 pt-5 mt-3">
        <div className={`toast ${showNotificationToast ? 'show' : ''}`}>
          <div className="toast-header">
            <strong className="me-auto">{t('notification_toast_title')}</strong>
            <button type="button" className="btn-close" data-bs-dismiss="toast" aria-label="Close" onClick={closeNotificationToast}></button>
          </div>
          <div className="toast-body">
            {t('notification_toast_description')}
            <div className="text-end">
              <button type="button" className="btn btn-primary btn-sm" onClick={requestNotification}>
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
