import React, {useEffect, useState} from 'react';
import {useNavigate} from 'react-router-dom';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import LoginForm from './components/login-form';
import LoginFormCard from './components/login-form-card';

import {Snackbar} from '@/components';
import {useAuth} from '@/utils/auth';

const LoginPage = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (loggedIn) {
      navigate('/launch', {replace: true});
    }
  }, [loggedIn]);

  return (
    <>
      <LoginFormCard title={t('title_login')}>
        <Snackbar onClose={() => setFeedback('')} message={feedback} />
        <LoginForm setFeedback={setFeedback} />
      </LoginFormCard>
      <Helmet>
        <title>{t('title_login')}</title>
      </Helmet>
    </>
  );
};

export default LoginPage;
