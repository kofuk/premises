import React, {useState} from 'react';

import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import KeyIcon from '@mui/icons-material/Key';
import {LoadingButton} from '@mui/lab';
import {Button, ButtonGroup, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Tooltip} from '@mui/material';

import {LoginResult, useAuth, usePasskeysSupported} from '@/utils/auth';

interface Prop {
  setFeedback: (feedback: string) => void;
}

const LoginForm = ({setFeedback}: Prop) => {
  const [t] = useTranslation();

  const loginForm = useForm();
  const resetPasswdForm = useForm();

  const [loggingIn, setLoggingIn] = useState(false);
  const [openResetPasswordDialog, setOpenResetPasswordDialog] = useState(false);

  const {login, loginPasskeys, initializePassword} = useAuth();

  const handleLogin = ({username, password}: any) => {
    (async () => {
      setLoggingIn(true);
      try {
        const result = await login(username, password);
        if (result === LoginResult.LoggedIn) {
          setLoggingIn(false);
        } else {
          setOpenResetPasswordDialog(true);
        }
      } catch (err: any) {
        setLoggingIn(false);
        setFeedback(err.message);
      }
    })();
  };

  const handlePasskeys = async () => {
    try {
      await loginPasskeys();
      setLoggingIn(false);
    } catch (err: any) {
      console.error(err);
      setLoggingIn(false);
      setFeedback(err.message);
    }
  };

  const passkeysSupported = usePasskeysSupported();

  const handleChangePassword = ({password}: any) => {
    (async () => {
      try {
        await initializePassword(loginForm.getValues('username'), password);
      } catch (err: any) {
        console.error(err);
        setFeedback(err.message);
      }
    })();
  };

  return (
    <>
      <form onSubmit={loginForm.handleSubmit(handleLogin)}>
        <Stack spacing={2}>
          <TextField autoComplete="username" fullWidth label={t('username')} type="text" variant="outlined" {...loginForm.register('username')} />
          <TextField autoComplete="password" fullWidth label={t('password')} type="password" variant="outlined" {...loginForm.register('password')} />
          <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
            <ButtonGroup aria-label="outlined primary button group" disabled={loggingIn} variant="contained">
              {passkeysSupported && (
                <Tooltip title="Use Passkey">
                  <Button aria-label="security key" onClick={handlePasskeys} size="small" type="button">
                    <KeyIcon />
                  </Button>
                </Tooltip>
              )}
              <LoadingButton loading={loggingIn} type="submit" variant="contained">
                {t('login')}
              </LoadingButton>
            </ButtonGroup>
          </Stack>
        </Stack>
      </form>
      <Dialog open={openResetPasswordDialog}>
        <DialogTitle>{t('set_password_title')}</DialogTitle>
        <form onSubmit={resetPasswdForm.handleSubmit(handleChangePassword)}>
          <DialogContent>
            <Stack spacing={2}>
              <TextField
                autoComplete="new-password"
                label={t('change_password_new')}
                type="password"
                {...resetPasswdForm.register('password', {
                  required: true
                })}
                fullWidth
              />
              <TextField
                autoComplete="new-password"
                label={t('change_password_confirm')}
                type="password"
                {...resetPasswdForm.register('passwordConfirm', {
                  required: true,
                  validate: (val: string) => {
                    if (resetPasswdForm.watch('password') !== val) {
                      return 'Password do not match';
                    }
                  }
                })}
                fullWidth
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button disabled={!resetPasswdForm.formState.isValid} type="submit">
              {t('set_password_submit')}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </>
  );
};

export default LoginForm;
