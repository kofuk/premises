import React from 'react';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import {Box} from '@mui/material';

import {ConfigProvider} from './components/launch-config';
import LaunchPage from './components/launch-page';
import LoadingPage from './components/loading-page';
import ManualSetupPage from './components/manual-setup-page';
import ServerControlPage from './components/server-control-page';

import {useRunnerStatus} from '@/utils/runner-status';

const PAGE_LAUNCH = 1;
const PAGE_LOADING = 2;
const PAGE_RUNNING = 3;
const PAGE_MANUAL_SETUP = 4;

const Launcher = () => {
  const [t] = useTranslation();

  const {pageCode: page} = useRunnerStatus();

  const createMainPane = (page: number) => {
    switch (page) {
      case PAGE_LAUNCH:
        return <LaunchPage />;
      case PAGE_LOADING:
        return <LoadingPage />;
      case PAGE_RUNNING:
        return <ServerControlPage />;
      case PAGE_MANUAL_SETUP:
        return <ManualSetupPage />;
      default:
        throw new Error(`Unkwnon page: ${page}`);
    }
  };

  const mainPane: React.ReactElement = createMainPane(page);
  return (
    <ConfigProvider>
      <Box sx={{maxWidth: 1000, m: '0 auto', p: 2}}>{mainPane}</Box>

      <Helmet>
        <title>{t('app_name')}</title>
      </Helmet>
    </ConfigProvider>
  );
};

export default Launcher;
