import {useState, useRef} from 'react';
import KeyIcon from '@mui/icons-material/Key';
import {DialogActions, DialogContent, DialogTitle, Dialog, ButtonGroup, Tooltip, Stack, Button, TextField} from '@mui/material';
import {LoadingButton} from '@mui/lab';
import '@/i18n';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {LoginResult, useAuth} from '@/utils/auth';
import {useForm} from 'react-hook-form';

interface PasswordLoginProps {
  setFeedback: (feedback: string) => void;
  switchToSecurityKey: () => void;
}

export default ({setFeedback, switchToSecurityKey}: PasswordLoginProps) => {
  const loginForm = useForm();
  const resetPasswdForm = useForm();

  const [loggingIn, setLoggingIn] = useState(false);
  const [openResetPasswordDialog, setOpenResetPasswordDialog] = useState(false);

  const navigate = useNavigate();

  const {login, initializePassword} = useAuth();

  const handleLogin = ({username, password}: any) => {
    setLoggingIn(true);

    login(username, password).then(
      (result) => {
        if (result === LoginResult.LoggedIn) {
          setLoggingIn(false);
          navigate('/launch', {replace: true});
        } else {
          setOpenResetPasswordDialog(true);
        }
      },
      (err) => {
        setLoggingIn(false);
        setFeedback(err.message);
      }
    );
  };

  const handleChangePassword = ({password}: any) => {
    initializePassword(loginForm.getValues('username'), password).then(
      () => {
        navigate('/launch', {replace: true});
      },
      (err) => {
        setFeedback(err.message);
      }
    );
  };

  return (
    <>
      <form onSubmit={loginForm.handleSubmit(handleLogin)}>
        <Stack spacing={2}>
          <TextField variant="outlined" label={t('username')} autoComplete="username" type="text" fullWidth {...loginForm.register('username')} />
          <TextField variant="outlined" label={t('password')} autoComplete="password" type="password" {...loginForm.register('password')} fullWidth />
          <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
            <ButtonGroup disabled={loggingIn} variant="contained" aria-label="outlined primary button group">
              <Tooltip title="Use security key">
                <Button size="small" aria-label="security key" type="button" onClick={() => switchToSecurityKey()}>
                  <KeyIcon />
                </Button>
              </Tooltip>
              <LoadingButton loading={loggingIn} variant="contained" type="submit">
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
                label={t('change_password_new')}
                type="password"
                autoComplete="new-password"
                {...resetPasswdForm.register('password', {
                  required: true
                })}
                fullWidth
              />
              <TextField
                label={t('change_password_confirm')}
                type="password"
                autoComplete="new-password"
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
