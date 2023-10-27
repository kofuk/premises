import React from 'react';
import {useRoutes} from 'react-router-dom';

const LoginPage = React.lazy(() => import('@/features/login'));
const LaunchPage = React.lazy(() => import('@/features/launch'));
const UI = React.lazy(() => import('@/features/ui'));
const SettingsRoutes = React.lazy(() => import('@/features/settings/routes'));

const AppRoutes = () => {
  const routes = useRoutes([
    {
      path: '/',
      element: <LoginPage />
    },
    {
      path: '/launch',
      element: <UI />,
      children: [
        {
          index: true,
          element: <LaunchPage />
        }
      ]
    },
    {
      path: '/settings/*',
      element: <UI />,
      children: [
        {
          path: '*',
          element: <SettingsRoutes />
        }
      ]
    }
  ]);

  return routes;
};

export default AppRoutes;
