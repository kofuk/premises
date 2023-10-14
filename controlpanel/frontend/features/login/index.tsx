import {useEffect, useState} from 'react';
import {Helmet} from 'react-helmet-async';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '@/utils/auth';
import LoginForm from './components/login-form';
import LoginFormCard from './components/login-form-card';
import Snackbar from '@/components/snackbar';

export default () => {
  const [feedback, setFeedback] = useState('');

  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (loggedIn) {
      navigate('/launch', {replace: true});
    }
  }, []);

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
