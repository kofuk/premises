import React, {useState} from 'react';

import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Box, Stack, TextField, Typography} from '@mui/material';

import {APIError, addUser} from '@/api';
import LoadingButtonWithResult from '@/components/loading-button-with-result';
import Snackbar from '@/components/snackbar';

const AddUser = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

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
        setFeedback(err.message);
      }
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <>
      <Typography variant="h4" sx={{mt: 3}}>
        {t('add_user_header')}
      </Typography>

      <form onSubmit={handleSubmit(handleAddUser)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                label={t('username')}
                variant="outlined"
                type="text"
                autoComplete="username"
                fullWidth
                {...register('userName', {
                  required: true,
                  validate: (val: string) => val.length <= 32
                })}
              />
              <Box sx={{width: '100%'}} />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                label={t('password')}
                variant="outlined"
                type="password"
                autoComplete="new-password"
                fullWidth
                {...register('password', {
                  required: true
                })}
              />
              <TextField
                label={t('password_confirm')}
                variant="outlined"
                type="password"
                autoComplete="new-password"
                fullWidth
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
          <LoadingButtonWithResult type="submit" variant="contained" disabled={!formState.isValid} loading={submitting} success={success}>
            {t('add_user_submit')}
          </LoadingButtonWithResult>
        </Box>
      </form>

      <Snackbar onClose={() => setFeedback('')} message={feedback} />

      <Helmet>
        <title>
          {t('add_user_header')} - {t('app_name')}
        </title>
      </Helmet>
    </>
  );
};

export default AddUser;
