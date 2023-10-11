import React, {Suspense} from 'react';
import {BrowserRouter} from 'react-router-dom';
import {HelmetProvider} from 'react-helmet-async';
import {Routes, Route} from 'react-router-dom';
import {AuthProvider} from './utils/auth';
import Loading from './components/loading';
const LaunchPage = React.lazy(() => import('@/features/launch'));
const LoginPage = React.lazy(() => import('@/features/login'));
const UI = React.lazy(() => import('@/features/ui'));
const Settings = React.lazy(() => import('@/features/settings'));

export default () => {
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
