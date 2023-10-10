import {Suspense} from 'react';
import '@/i18n';
import {t} from 'i18next';
import Settings from './settings';
import {useNavigate, Outlet} from 'react-router-dom';
import {useAuth} from '@/utils/auth';
import Loading from '@/components/loading';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import 'bootstrap/js/dist/offcanvas';
import 'bootstrap/js/dist/collapse';
import 'bootstrap/scss/bootstrap.scss';
/////

export default () => {
  const navigate = useNavigate();
  const {logout} = useAuth();
  const handleLogout = () => {
    logout().then(() => {
      navigate('/', {replace: true});
    });
  };

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
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </div>
    </>
  );
};
