import {lazy} from 'react';
import {useRoutes} from 'react-router-dom';

const LoginPage = lazy(() => import('@/features/login'));
const LaunchPage = lazy(() => import('@/features/launch'));
const UI = lazy(() => import('@/features/ui'));
const SettingsRoutes = lazy(() => import('@/features/settings/routes'));

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
