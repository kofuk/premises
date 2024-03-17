import React, {Suspense, useEffect} from 'react';
import {Outlet, useNavigate} from 'react-router-dom';

import Navbar from './components/navbar';

import {Loading, StatusCollector} from '@/components';
import {useAuth} from '@/utils/auth';

const UI = () => {
  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (!loggedIn) {
      navigate('/', {replace: true});
    }
  }, [loggedIn]);

  return (
    loggedIn && (
      <>
        <Navbar />

        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>

        <StatusCollector />
      </>
    )
  );
};

export default UI;
