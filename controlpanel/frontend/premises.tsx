import React, {Suspense} from 'react';
import {BrowserRouter as Router} from 'react-router-dom';

import {SnackbarProvider} from 'notistack';
import {HelmetProvider} from 'react-helmet-async';

import {Loading, StatusCollector} from './components';
import AppRoutes from './routes';
import {AuthProvider} from './utils/auth';
import {RunnerStatusProvider} from './utils/runner-status';

import './i18n';

const Premises = () => {
  return (
    <HelmetProvider>
      <AuthProvider>
        <SnackbarProvider anchorOrigin={{horizontal: 'right', vertical: 'top'}}>
          <RunnerStatusProvider>
            <StatusCollector />
            <Router>
              <Suspense fallback={<Loading />}>
                <AppRoutes />
              </Suspense>
            </Router>
          </RunnerStatusProvider>
        </SnackbarProvider>
      </AuthProvider>
    </HelmetProvider>
  );
};

export default Premises;
