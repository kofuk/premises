import {useState} from 'react';
import PasswordIcon from '@mui/icons-material/Password';
import {ButtonGroup, Tooltip, Stack, Button, TextField} from '@mui/material';
import {LoadingButton} from '@mui/lab';
import {useForm} from 'react-hook-form';
import '@/i18n';
import {t} from 'i18next';
import {useNavigate} from 'react-router-dom';
import {useAuth} from '@/utils/auth';

interface WebAuthnLoginProps {
  setFeedback: (feedback: string) => void;
  switchToPassword: () => void;
}

export default ({setFeedback, switchToPassword}: WebAuthnLoginProps) => {
  const {register, handleSubmit, formState} = useForm();

  const [loggingIn, setLoggingIn] = useState(false);

  const navigate = useNavigate();

  const {loginWebAuthn} = useAuth();

  const handleWebAuthn = async ({username}: any) => {
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
    <form onSubmit={handleSubmit(handleWebAuthn)}>
      <Stack spacing={2}>
        <TextField variant="outlined" label={t('username')} autoComplete="username" type="text" fullWidth {...register('username')} />
        <Stack direction="row" justifyContent="end" sx={{mt: 1}}>
          <ButtonGroup disabled={loggingIn} variant="contained" aria-label="outlined primary button group">
            <Tooltip title="Use password">
              <Button size="small" aria-label="password login" type="button" onClick={() => switchToPassword()}>
                <PasswordIcon />
              </Button>
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
