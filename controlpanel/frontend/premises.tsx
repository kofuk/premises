import React, {Suspense} from 'react';
import {BrowserRouter as Router} from 'react-router-dom';

import {SnackbarKey, SnackbarProvider, closeSnackbar} from 'notistack';
import {HelmetProvider} from 'react-helmet-async';

import {Close as CloseIcon} from '@mui/icons-material';
import {IconButton} from '@mui/material';

import {Loading} from './components';
import AppRoutes from './routes';
import {AuthProvider} from './utils/auth';
import {RunnerStatusProvider} from './utils/runner-status';

import './i18n';

const Premises = () => {
  return (
    <HelmetProvider>
      <AuthProvider>
        <SnackbarProvider
          action={(key: SnackbarKey) => (
            <IconButton onClick={() => closeSnackbar(key)}>
              <CloseIcon sx={{color: 'white'}} />
            </IconButton>
          )}
          anchorOrigin={{horizontal: 'left', vertical: 'top'}}
          disableWindowBlurListener={true}
        >
          <RunnerStatusProvider>
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
