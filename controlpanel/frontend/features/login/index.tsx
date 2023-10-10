import {useEffect, useState} from 'react';
import {Helmet} from 'react-helmet-async';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '@/utils/auth';
import WebAuthnLoginForm from './components/webauthn-login-form';
import PasswordLoginForm from './components/password-login-form';
import LoginFormCard from './components/login-form-card';
import Snackbar from '@/components/snackbar';

export default () => {
  const [feedback, setFeedback] = useState('');
  const [loginMethod, setLoginMethod] = useState('password');

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
        {loginMethod === 'password' ? (
          <PasswordLoginForm setFeedback={setFeedback} switchToSecurityKey={() => setLoginMethod('webauthn')} />
        ) : (
          <WebAuthnLoginForm setFeedback={setFeedback} switchToPassword={() => setLoginMethod('password')} />
        )}
      </LoginFormCard>
      <Helmet>
        <title>{t('title_login')}</title>
      </Helmet>
    </>
  );
};
