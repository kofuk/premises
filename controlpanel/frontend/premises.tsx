import * as React from 'react';
import {BrowserRouter} from 'react-router-dom';
import {HelmetProvider} from 'react-helmet-async';
import {Routes, Route} from 'react-router-dom';
import App from './control/app';
import Login from './login/login';

export default () => {
  return (
    <React.StrictMode>
      <HelmetProvider>
        <BrowserRouter>
          <Routes>
            <Route index element={<Login />} />
            <Route path="/launch" element={<App />} />
          </Routes>
        </BrowserRouter>
      </HelmetProvider>
    </React.StrictMode>
  );
};
