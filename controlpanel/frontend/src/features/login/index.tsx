import {Box} from '@mui/material';
import {useEffect} from 'react';

import {useTranslation} from 'react-i18next';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '@/utils/auth';
import LoginForm from './components/login-form';

const LoginPage = () => {
  const [t] = useTranslation();

  const navigate = useNavigate();
  const {loggedIn} = useAuth();
  useEffect(() => {
    if (loggedIn) {
      navigate('/launch', {replace: true});
    }
  }, [loggedIn, navigate]);

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
