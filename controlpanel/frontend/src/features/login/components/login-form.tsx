import {useState} from 'react';

import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

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
        toast.error(err.message);
      }
    })();
  };

  const handleChangePassword = ({password}: any) => {
    (async () => {
      try {
        await initializePassword(loginForm.getValues('username'), password);
      } catch (err: any) {
        toast.error(err.message);
      }
    })();
  };

  return (
    <Box display="flex" justifyContent="center">
      <Card sx={{minWidth: 350, p: 3, mt: 5}} variant="outlined">
        <CardContent>
          <Typography component="h1" sx={{mb: 3}} variant="h4">
            {t('login.title')}
          </Typography>
          <form onSubmit={loginForm.handleSubmit(handleLogin)}>
            <Stack spacing={2}>
              <TextField
                autoComplete="username"
                fullWidth
                label={t('login.username')}
                type="text"
                variant="outlined"
                {...loginForm.register('username')}
              />
              <TextField
                autoComplete="password"
                fullWidth
                label={t('login.password')}
                type="password"
                variant="outlined"
                {...loginForm.register('password')}
              />
              <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
                <LoadingButton loading={loggingIn} type="submit" variant="contained">
                  {t('login.login')}
                </LoadingButton>
              </Stack>
            </Stack>
          </form>
          <Dialog open={openResetPasswordDialog}>
            <DialogTitle>{t('login.change_password')}</DialogTitle>
            <form onSubmit={resetPasswdForm.handleSubmit(handleChangePassword)}>
              <DialogContent>
                <Stack spacing={2}>
                  <TextField
                    autoComplete="new-password"
                    label={t('login.change_password.password')}
                    type="password"
                    {...resetPasswdForm.register('password', {
                      required: true
                    })}
                    fullWidth
                  />
                  <TextField
                    autoComplete="new-password"
                    label={t('login.change_password.confirm')}
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
                  {t('login.change_password.save')}
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
