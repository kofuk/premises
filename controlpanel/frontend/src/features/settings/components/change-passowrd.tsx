import {useState} from 'react';

import {useSnackbar} from 'notistack';
import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Paper, Stack, TextField, Typography} from '@mui/material';
import {Box} from '@mui/system';

import {changePassword} from '@/api';
import LoadingButtonWithResult from '@/components/loading-button-with-result';
import {useAuth} from '@/utils/auth';

const ChangePassword = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const {register, handleSubmit, formState, watch} = useForm();
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const {enqueueSnackbar} = useSnackbar();

  const handleChangePassword = ({currentPassword, newPassword}: any) => {
    (async () => {
      setSubmitting(true);
      setSuccess(false);
      try {
        await changePassword(accessToken, {password: currentPassword, newPassword});
        setSuccess(true);
      } catch (err: any) {
        console.error(err);
        enqueueSnackbar(err.message, {variant: 'error'});
      } finally {
        setSubmitting(false);
      }
    })();
  };

  return (
    <Paper variant="outlined">
      <Typography sx={{mt: 3}} variant="h4">
        {t('settings.change_password')}
      </Typography>

      <form onSubmit={handleSubmit(handleChangePassword)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="current-password"
                fullWidth
                label={t('settings.change_password.current')}
                type="password"
                variant="outlined"
                {...register('currentPassword', {
                  required: true
                })}
              />
              <Box sx={{width: '100%'}} />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('settings.change_password.new')}
                type="password"
                variant="outlined"
                {...register('newPassword', {
                  required: true
                })}
              />
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('settings.change_password.confirm')}
                type="password"
                variant="outlined"
                {...register('newPasswordConfirm', {
                  required: true,
                  validate: (val: string) => {
                    if (watch('newPassword') !== val) {
                      return 'Password do not match';
                    }
                  }
                })}
              />
            </Stack>
          </Stack>
        </Box>

        <Box sx={{m: 2}}>
          <LoadingButtonWithResult disabled={!formState.isValid} loading={submitting} success={success} type="submit" variant="contained">
            {t('settings.change_password.save')}
          </LoadingButtonWithResult>
        </Box>
      </form>

      <Helmet>
        <title>
          {t('settings.title')} - {t('app_name')}
        </title>
      </Helmet>
    </Paper>
  );
};

export default ChangePassword;
