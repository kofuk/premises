import React from 'react';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import LoadingPage from './components/loading-page';
import ServerConfigPane from './server-config-pane';
import ServerControlPane from './server-control-pane';

import {useRunnerStatus} from '@/utils/runner-status';

// For bootstrap based screen. We should remove this after migrating to styled-component completed.
import 'bootstrap/scss/bootstrap.scss';
/////

const PAGE_LAUNCH = 1;
const PAGE_LOADING = 2;
const PAGE_RUNNING = 3;

const LaunchPage = () => {
  const [t] = useTranslation();

  const {pageCode: page} = useRunnerStatus();

  const createMainPane = (page: number) => {
    if (page == PAGE_LAUNCH) {
      return <ServerConfigPane />;
    } else if (page == PAGE_LOADING) {
      return <LoadingPage />;
    } else if (page == PAGE_RUNNING) {
      return <ServerControlPane />;
    }
    throw new Error(`Unkwnon page: ${page}`);
  };

  const mainPane: React.ReactElement = createMainPane(page);
  return (
    <>
      {mainPane}

      <Helmet>
        <title>{t('app_name')}</title>
      </Helmet>
    </>
  );
};

export default LaunchPage;
