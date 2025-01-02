import {useEffect} from 'react';
import {useNavigate} from 'react-router-dom';

import {useTranslation} from 'react-i18next';

import {Box} from '@mui/material';

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
      <Box sx={{maxWidth: 1000, m: '0 auto', p: 2}}>
        <LoginForm />
      </Box>
      <title>{t('login.title')}</title>
    </>
  );
};

export default LoginPage;
