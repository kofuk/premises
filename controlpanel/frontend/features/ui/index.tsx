import React, {Suspense} from 'react';
import {Outlet} from 'react-router-dom';

import Navbar from './components/navbar';

import Loading from '@/components/loading';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import 'bootstrap/js/dist/collapse';
import 'bootstrap/scss/bootstrap.scss';
/////

const UI = () => {
  return (
    <>
      <Navbar />

      <div className="container">
        <Suspense fallback={<Loading />}>
          <Outlet />
        </Suspense>
      </div>
    </>
  );
};

export default UI;
