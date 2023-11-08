import React, {Suspense} from 'react';
import {BrowserRouter as Router} from 'react-router-dom';

import {SnackbarProvider} from 'notistack';
import {HelmetProvider} from 'react-helmet-async';

import {Loading} from './components';
import AppRoutes from './routes';
import {AuthProvider} from './utils/auth';

import './i18n';

const Premises = () => {
  return (
    <HelmetProvider>
      <AuthProvider>
        <SnackbarProvider anchorOrigin={{horizontal: 'right', vertical: 'top'}}>
          <Router>
            <Suspense fallback={<Loading />}>
              <AppRoutes />
            </Suspense>
          </Router>
        </SnackbarProvider>
      </AuthProvider>
    </HelmetProvider>
  );
};

export default Premises;
