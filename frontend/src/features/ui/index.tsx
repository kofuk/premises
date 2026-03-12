import {Suspense, useEffect} from 'react';
import {Outlet, useNavigate} from 'react-router-dom';
import Loading from '@/components/loading';
import StatusCollector from '@/components/status-collector';
import {useAuth} from '@/utils/auth';
import Navbar from './components/navbar';

const UI = () => {
  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (!loggedIn) {
      navigate('/', {replace: true});
    }
  }, [loggedIn, navigate]);

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
