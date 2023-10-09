import React, {useEffect, useState} from 'react';
import KeyIcon from '@mui/icons-material/Key';
import CloseIcon from '@mui/icons-material/Close';
import PasswordIcon from '@mui/icons-material/Password';
import {
  DialogActions,
  DialogContent,
  DialogTitle,
  Dialog,
  IconButton,
  ButtonGroup,
  Tooltip,
  Box,
  Stack,
  Button,
  Card,
  Typography,
  CardContent,
  TextField,
  Snackbar
} from '@mui/material';
import {LoadingButton} from '@mui/lab';
import {Helmet} from 'react-helmet-async';

import '@/i18n';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {LoginResult, useAuth} from '@/utils/auth';

interface WebAuthnLoginProps {
  setFeedback: (feedback: string) => void;
  switchToPassword: () => void;
}

const WebAuthnLogin: React.FC<WebAuthnLoginProps> = ({setFeedback, switchToPassword}: WebAuthnLoginProps) => {
  const [username, setUsername] = useState('');
  const [loggingIn, setLoggingIn] = useState(false);

  const navigate = useNavigate();

  const {loginWebAuthn} = useAuth();

  const handleWebAuthn = async () => {
    loginWebAuthn(username).then(
      () => {
        setLoggingIn(false);
        navigate('/launch', {replace: true});
      },
      (err) => {
        setLoggingIn(false);
        setFeedback(err.message);
      }
    );
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        handleWebAuthn();
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
        <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
          <ButtonGroup disabled={loggingIn} variant="contained" aria-label="outlined primary button group">
            <Tooltip title="Use password">
              <Button aria-label="password login" startIcon={<PasswordIcon />} type="button" onClick={() => switchToPassword()} />
            </Tooltip>
            <LoadingButton loading={loggingIn} variant="contained" type="submit">
              {t('login')}
            </LoadingButton>
          </ButtonGroup>
        </Stack>
      </Stack>
    </form>
  );
};

interface PasswordLoginProps {
  setFeedback: (feedback: string) => void;
  switchToSecurityKey: () => void;
}

const PasswordLogin: React.FC<PasswordLoginProps> = ({setFeedback, switchToSecurityKey}: PasswordLoginProps) => {
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
                <Button aria-label="security key" startIcon={<KeyIcon />} type="button" onClick={() => switchToSecurityKey()} />
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
    <Box display="flex" justifyContent="center">
      <Card sx={{minWidth: 350, p: 3, mt: 5}}>
        <CardContent>
          <Typography variant="h4" component="h1" sx={{mb: 3}}>
            {t('title_login')}
          </Typography>
          <Snackbar
            anchorOrigin={{vertical: 'top', horizontal: 'center'}}
            open={feedback.length > 0}
            autoHideDuration={10000}
            onClose={() => setFeedback('')}
            message={feedback}
            action={
              <>
                <IconButton aria-label="close" color="inherit" sx={{p: 0.5}} onClick={() => setFeedback('')}>
                  <CloseIcon />
                </IconButton>
              </>
            }
          />
          {loginMethod === 'password' ? (
            <PasswordLogin setFeedback={setFeedback} switchToSecurityKey={() => setLoginMethod('webauthn')} />
          ) : (
            <WebAuthnLogin setFeedback={setFeedback} switchToPassword={() => setLoginMethod('password')} />
          )}
        </CardContent>
      </Card>
      <Helmet>
        <title>{t('title_login')}</title>
      </Helmet>
    </Box>
  );
};
