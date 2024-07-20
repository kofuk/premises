import {Suspense, useEffect} from 'react';
import {Outlet, useNavigate} from 'react-router-dom';

import Navbar from './components/navbar';

import Loading from '@/components/loading';
import StatusCollector from '@/components/status-collector';
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
