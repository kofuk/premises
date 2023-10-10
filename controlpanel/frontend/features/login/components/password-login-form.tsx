import {useState} from 'react';
import KeyIcon from '@mui/icons-material/Key';
import {DialogActions, DialogContent, DialogTitle, Dialog, ButtonGroup, Tooltip, Stack, Button, TextField} from '@mui/material';
import {LoadingButton} from '@mui/lab';
import '@/i18n';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {LoginResult, useAuth} from '@/utils/auth';

interface PasswordLoginProps {
  setFeedback: (feedback: string) => void;
  switchToSecurityKey: () => void;
}

export default ({setFeedback, switchToSecurityKey}: PasswordLoginProps) => {
  const [loggingIn, setLoggingIn] = useState(false);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const [openResetPasswordDialog, setOpenResetPasswordDialog] = useState(false);

  const navigate = useNavigate();

  const {login, initializePassword} = useAuth();

  const handleLogin = () => {
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

  const [newPassword, setNewPassword] = useState('');
  const [newPasswordConfirm, setNewPasswordConfirm] = useState('');

  const handleChangePassword = () => {
    initializePassword(username, newPassword).then(
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
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleLogin();
        }}
      >
        <Stack spacing={2}>
          <TextField
            variant="outlined"
            label={t('username')}
            autoComplete="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            fullWidth
          />
          <TextField
            variant="outlined"
            label={t('password')}
            autoComplete="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            fullWidth
          />
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
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleChangePassword();
          }}
        >
          <DialogContent>
            <Stack spacing={2}>
              <TextField
                label={t('change_password_new')}
                type="password"
                autoComplete="new-password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                fullWidth
              />
              <TextField
                label={t('change_password_confirm')}
                type="password"
                autoComplete="new-password"
                value={newPasswordConfirm}
                onChange={(e) => setNewPasswordConfirm(e.target.value)}
                fullWidth
              />
            </Stack>
          </DialogContent>
          <DialogActions>
            <Button disabled={!(newPassword.length != 0 && newPassword == newPasswordConfirm)} type="submit">
              {t('set_password_submit')}
            </Button>
          </DialogActions>
        </form>
      </Dialog>
    </>
  );
};
