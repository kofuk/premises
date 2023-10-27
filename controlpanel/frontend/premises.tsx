import React, {Suspense} from 'react';
import {BrowserRouter as Router} from 'react-router-dom';

import {HelmetProvider} from 'react-helmet-async';

import Loading from './components/loading';
import AppRoutes from './routes';
import {AuthProvider} from './utils/auth';

import './i18n';

const Premises = () => {
  return (
    <HelmetProvider>
      <AuthProvider>
        <Router>
          <Suspense fallback={<Loading />}>
            <AppRoutes />
          </Suspense>
        </Router>
      </AuthProvider>
    </HelmetProvider>
  );
};

export default Premises;
