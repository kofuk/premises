import {Suspense} from 'react';
import '@/i18n';
import {t} from 'i18next';
import {useNavigate, Outlet, Link} from 'react-router-dom';
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
            <Link to="/settings" className="btn btn-link me-1">
              {t('settings')}
            </Link>
            <button onClick={handleLogout} className="btn btn-primary bg-gradient">
              {t('logout')}
            </button>
          </div>
        </div>
      </nav>

      <div className="container">
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </div>
    </>
  );
};
