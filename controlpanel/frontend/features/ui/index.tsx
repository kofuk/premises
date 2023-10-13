import {Suspense} from 'react';
import Loading from '@/components/loading';
import Navbar from './components/navbar';
import {Outlet} from 'react-router-dom';

// For bootstrap based screen. We should remove this after transition to styled-component completed.
import 'bootstrap/js/dist/offcanvas';
import 'bootstrap/js/dist/collapse';
import 'bootstrap/scss/bootstrap.scss';
/////

export default () => {
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
