import {Suspense} from 'react';
import {BrowserRouter as Router} from 'react-router-dom';

import {HelmetProvider} from 'react-helmet-async';
import {ToastContainer} from 'react-toastify';

import Loading from './components/loading';
import AppRoutes from './routes';
import {AuthProvider} from './utils/auth';
import {RunnerStatusProvider} from './utils/runner-status';

import './i18n';

const Premises = () => {
  return (
    <HelmetProvider>
      <AuthProvider>
        <ToastContainer />
        <RunnerStatusProvider>
          <Router>
            <Suspense fallback={<Loading />}>
              <AppRoutes />
            </Suspense>
          </Router>
        </RunnerStatusProvider>
      </AuthProvider>
    </HelmetProvider>
  );
};

export default Premises;
