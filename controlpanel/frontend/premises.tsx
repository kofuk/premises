import React from 'react';
import {BrowserRouter} from 'react-router-dom';
import {HelmetProvider} from 'react-helmet-async';
import {Routes, Route} from 'react-router-dom';
import LaunchPage from '@/features/launch';
import LoginPage from '@/features/login';
import {AuthProvider} from './utils/auth';

export default () => {
  return (
    <React.StrictMode>
      <AuthProvider>
        <HelmetProvider>
          <BrowserRouter>
            <Routes>
              <Route index element={<LoginPage />} />
              <Route path="/launch" element={<LaunchPage />} />
            </Routes>
          </BrowserRouter>
        </HelmetProvider>
      </AuthProvider>
    </React.StrictMode>
  );
};
