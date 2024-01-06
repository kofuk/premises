import React, {useState} from 'react';

import {useSnackbar} from 'notistack';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {LoadingButton} from '@mui/lab';
import {Box, Button, Card, CardContent, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField, Typography} from '@mui/material';

import {LoginResult, useAuth} from '@/utils/auth';

const LoginForm = () => {
  const [t] = useTranslation();

  const loginForm = useForm();
  const resetPasswdForm = useForm();

  const [loggingIn, setLoggingIn] = useState(false);
  const [openResetPasswordDialog, setOpenResetPasswordDialog] = useState(false);

  const {login, initializePassword} = useAuth();

  const {enqueueSnackbar} = useSnackbar();

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
        enqueueSnackbar(err.message, {variant: 'error'});
      }
    })();
  };

  const handleChangePassword = ({password}: any) => {
    (async () => {
      try {
        await initializePassword(loginForm.getValues('username'), password);
      } catch (err: any) {
        console.error(err);
        enqueueSnackbar(err.message, {variant: 'error'});
      }
    })();
  };

  return (
    <Box display="flex" justifyContent="center">
      <Card sx={{minWidth: 350, p: 3, mt: 5}}>
        <CardContent>
          <Typography component="h1" sx={{mb: 3}} variant="h4">
            {t('title_login')}
          </Typography>
          <form onSubmit={loginForm.handleSubmit(handleLogin)}>
            <Stack spacing={2}>
              <TextField autoComplete="username" fullWidth label={t('username')} type="text" variant="outlined" {...loginForm.register('username')} />
              <TextField
                autoComplete="password"
                fullWidth
                label={t('password')}
                type="password"
                variant="outlined"
                {...loginForm.register('password')}
              />
              <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
                <LoadingButton loading={loggingIn} type="submit" variant="contained">
                  {t('login')}
                </LoadingButton>
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
        </CardContent>
      </Card>
    </Box>
  );
};

export default LoginForm;
