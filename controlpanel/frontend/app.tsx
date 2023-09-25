import * as React from 'react';
import {createRoot} from 'react-dom/client';
import {Routes, BrowserRouter, Route} from 'react-router-dom';
import Login from './login/login';
import App from './control/app';
import {HelmetProvider} from 'react-helmet-async';

// For material UI
import '@fontsource/roboto/300.css';
import '@fontsource/roboto/400.css';
import '@fontsource/roboto/500.css';
import '@fontsource/roboto/700.css';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import './control.scss';
import 'bootstrap/js/dist/offcanvas';
import 'bootstrap/js/dist/collapse';
/////

createRoot(document.getElementById('app')!).render(
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
