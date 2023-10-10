import React, {useState, useEffect} from 'react';

import '@/i18n';
import {t} from 'i18next';

import StatusBar from './statusbar';
import ServerControlPane from './server-control-pane';
import ServerConfigPane from './server-config-pane';
import Settings from './settings';
import {useNavigate} from 'react-router-dom';
import {Helmet} from 'react-helmet-async';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import './control.scss';
import 'bootstrap/js/dist/offcanvas';
import 'bootstrap/js/dist/collapse';
import {useAuth} from '@/utils/auth';
/////

export default () => {
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

  const navigate = useNavigate();
  const {loggedIn, logout} = useAuth();
  useEffect(() => {
    if (!loggedIn) {
      navigate('/', {replace: true});
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
    Notification.requestPermission().then((result) => {
      const granted = result === 'granted';
      setUseNotification(granted);
      if (granted) {
        closeNotificationToast();
      }
    });
  };

  const handleLogout = () => {
    logout().then(() => {
      navigate('/', {replace: true});
    });
  };

  const mainPane: React.ReactElement = isServerShutdown ? <ServerConfigPane showError={showError} /> : <ServerControlPane showError={showError} />;
  return (
    <>
      <nav className="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
        <div className="container-fluid">
          <span className="navbar-brand">{t('app_name')}</span>
          <div className="collapse navbar-collapse">
            <div className="navbar-nav me-auto"></div>
            <a className="btn btn-link me-1" data-bs-toggle="offcanvas" href="#settingsPane" role="button" aria-controls="settingsPane">
              {t('settings')}
            </a>
            <button onClick={handleLogout} className="btn btn-primary bg-gradient">
              {t('logout')}
            </button>
          </div>
        </div>
      </nav>

      <div className="offcanvas offcanvas-start" tabIndex={-1} id="settingsPane" aria-labelledby="SettingsLabel">
        <div className="offcanvas-header">
          <h5 className="offcanvas-title" id="settingsLabel">
            {t('settings')}
          </h5>
          <button type="button" className="btn-close text-reset" data-bs-dismiss="offcanvas" aria-label="Close"></button>
        </div>
        <div className="offcanvas-body">
          <Settings />
        </div>
      </div>

      <div className="container">
        <StatusBar isError={isError} message={message} />
        {mainPane}
      </div>

      <div className="toast-container position-absolute top-0 end-0 pe-1 pt-5">
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
