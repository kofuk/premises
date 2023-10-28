import React, {Suspense, useEffect} from 'react';
import {Outlet, useNavigate} from 'react-router-dom';

import {Box} from '@mui/material';

import Navbar from './components/navbar';

import Loading from '@/components/loading';
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

        <Box sx={{maxWidth: 1000, m: '0 auto', p: 2}}>
          <Suspense fallback={<Loading />}>
            <Outlet />
          </Suspense>
        </Box>
      </>
    )
  );
};

export default UI;
