import React, {useState} from 'react';

import {useSnackbar} from 'notistack';
import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Stack, TextField, Typography} from '@mui/material';
import {Box} from '@mui/system';

import {changePassword} from '@/api';
import {LoadingButtonWithResult} from '@/components';

const ChangePassword = () => {
  const [t] = useTranslation();

  const {register, handleSubmit, formState, watch} = useForm();
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);
  const {enqueueSnackbar} = useSnackbar();

  const handleChangePassword = ({currentPassword, newPassword}: any) => {
    (async () => {
      setSubmitting(true);
      setSuccess(false);
      try {
        await changePassword({password: currentPassword, newPassword});
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
    <>
      <Typography sx={{mt: 3}} variant="h4">
        {t('change_password_header')}
      </Typography>

      <form onSubmit={handleSubmit(handleChangePassword)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="current-password"
                fullWidth
                label={t('change_password_current')}
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
                label={t('change_password_new')}
                type="password"
                variant="outlined"
                {...register('newPassword', {
                  required: true
                })}
              />
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('change_password_confirm')}
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
            {t('change_password_submit')}
          </LoadingButtonWithResult>
        </Box>
      </form>

      <Helmet>
        <title>
          {t('change_password_header')} - {t('app_name')}
        </title>
      </Helmet>
    </>
  );
};

export default ChangePassword;
