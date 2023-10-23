import React, {Suspense, useEffect} from 'react';
import {Outlet, useNavigate} from 'react-router-dom';

import Navbar from './components/navbar';

import Loading from '@/components/loading';
import {useAuth} from '@/utils/auth';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import 'bootstrap/js/dist/collapse';
import 'bootstrap/scss/bootstrap.scss';
/////

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

        <div className="container">
          <Suspense fallback={<Loading />}>
            <Outlet />
          </Suspense>
        </div>
      </>
    )
  );
};

export default UI;
