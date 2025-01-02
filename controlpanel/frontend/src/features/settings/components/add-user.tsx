import {useState} from 'react';

import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';
import {toast} from 'react-toastify';

import {Box, Paper, Stack, TextField, Typography} from '@mui/material';

import {APIError, addUser} from '@/api';
import LoadingButtonWithResult from '@/components/loading-button-with-result';
import {useAuth} from '@/utils/auth';

const AddUser = () => {
  const [t] = useTranslation();

  const {accessToken} = useAuth();

  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  const {register, handleSubmit, formState, watch} = useForm();

  const handleAddUser = async ({userName, password}: any) => {
    setSubmitting(true);
    setSuccess(false);

    try {
      await addUser(accessToken, {userName, password});
      setSuccess(true);
    } catch (err: unknown) {
      console.error(err);
      if (err instanceof APIError) {
        toast.error(err.message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Paper variant="outlined">
      <Typography sx={{mt: 3}} variant="h4">
        {t('settings.add_user')}
      </Typography>

      <form onSubmit={handleSubmit(handleAddUser)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="username"
                fullWidth
                label={t('settings.add_user.username')}
                type="text"
                variant="outlined"
                {...register('userName', {
                  required: true,
                  validate: (val: string) => val.length <= 32
                })}
              />
              <Box sx={{width: '100%'}} />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('settings.add_user.password')}
                type="password"
                variant="outlined"
                {...register('password', {
                  required: true
                })}
              />
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('settings.add_user.password_confirm')}
                type="password"
                variant="outlined"
                {...register('passwordConfirm', {
                  required: true,
                  validate: (val: string) => {
                    if (watch('password') !== val) {
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
            {t('settings.add_user.save')}
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

export default AddUser;
