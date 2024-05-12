import React, {useState} from 'react';

import {useSnackbar} from 'notistack';
import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Box, Paper, Stack, TextField, Typography} from '@mui/material';

import {APIError, addUser} from '@/api';
import LoadingButtonWithResult from '@/components/loading-button-with-result';

const AddUser = () => {
  const [t] = useTranslation();

  const {enqueueSnackbar} = useSnackbar();

  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  const {register, handleSubmit, formState, watch} = useForm();

  const handleAddUser = async ({userName, password}: any) => {
    setSubmitting(true);
    setSuccess(false);

    try {
      await addUser({userName, password});
      setSuccess(true);
    } catch (err: unknown) {
      console.error(err);
      if (err instanceof APIError) {
        enqueueSnackbar(err.message, {variant: 'error'});
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Paper variant="outlined">
      <Typography sx={{mt: 3}} variant="h4">
        {t('add_user_header')}
      </Typography>

      <form onSubmit={handleSubmit(handleAddUser)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                autoComplete="username"
                fullWidth
                label={t('username')}
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
                label={t('password')}
                type="password"
                variant="outlined"
                {...register('password', {
                  required: true
                })}
              />
              <TextField
                autoComplete="new-password"
                fullWidth
                label={t('password_confirm')}
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
            {t('add_user_submit')}
          </LoadingButtonWithResult>
        </Box>
      </form>

      <Helmet>
        <title>
          {t('add_user_header')} - {t('app_name')}
        </title>
      </Helmet>
    </Paper>
  );
};

export default AddUser;
