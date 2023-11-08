import React, {useEffect} from 'react';
import {useNavigate} from 'react-router-dom';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import LoginForm from './components/login-form';

import {useAuth} from '@/utils/auth';

const LoginPage = () => {
  const [t] = useTranslation();

  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (loggedIn) {
      navigate('/launch', {replace: true});
    }
  }, [loggedIn]);

  return (
    <>
      <LoginForm />
      <Helmet>
        <title>{t('title_login')}</title>
      </Helmet>
    </>
  );
};

export default LoginPage;
