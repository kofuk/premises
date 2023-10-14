import React, {Suspense} from 'react';
import {BrowserRouter, Routes, Route} from 'react-router-dom';

import {HelmetProvider} from 'react-helmet-async';

import Loading from './components/loading';
import {AuthProvider} from './utils/auth';
const LaunchPage = React.lazy(() => import('@/features/launch'));
const LoginPage = React.lazy(() => import('@/features/login'));
const UI = React.lazy(() => import('@/features/ui'));
const Settings = React.lazy(() => import('@/features/settings'));

import './i18n';

const Premises = () => {
  return (
    <React.StrictMode>
      <HelmetProvider>
        <AuthProvider>
          <BrowserRouter>
            <Suspense fallback={<Loading />}>
              <Routes>
                <Route index element={<LoginPage />} />
                <Route path="/launch" element={<UI />}>
                  <Route index element={<LaunchPage />} />
                </Route>
                <Route path="/settings" element={<UI />}>
                  <Route index element={<Settings />} />
                </Route>
              </Routes>
            </Suspense>
          </BrowserRouter>
        </AuthProvider>
      </HelmetProvider>
    </React.StrictMode>
  );
};

export default Premises;
