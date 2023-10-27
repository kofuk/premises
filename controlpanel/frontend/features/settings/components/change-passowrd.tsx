import React, {useState} from 'react';

import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Check as CheckIcon} from '@mui/icons-material';
import {LoadingButton} from '@mui/lab';
import {Stack, TextField, Typography} from '@mui/material';
import {green} from '@mui/material/colors';
import {Box} from '@mui/system';

import Snackbar from '@/components/snackbar';

const ChangePassword = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const {register, handleSubmit, formState, watch} = useForm();
  const [submitting, setSubmitting] = useState(false);
  const [success, setSuccess] = useState(false);

  const handleChangePassword = ({currentPassword, newPassword}: any) => {
    (async () => {
      setSubmitting(true);
      setSuccess(false);

      const params = new URLSearchParams();
      params.append('password', currentPassword);
      params.append('new-password', newPassword);

      try {
        const result = await fetch('/api/users/change-password', {
          method: 'post',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          body: params.toString()
        }).then((resp) => resp.json());
        if (result['success']) {
          setSuccess(true);
          setFeedback('');
        } else {
          setSuccess(false);
          setFeedback(result['reason']);
        }
      } catch (err) {
        console.error(err);
      } finally {
        setSubmitting(false);
      }
    })();
  };

  const buttonSx = {
    ...(success && {
      bgcolor: green[500],
      '&:hover': {
        bgcolor: green[700]
      }
    })
  };

  return (
    <>
      <Typography variant="h4" sx={{mt: 3}}>
        {t('change_password_header')}
      </Typography>

      <form onSubmit={handleSubmit(handleChangePassword)}>
        <Box sx={{m: 2}}>
          <Stack spacing={2}>
            <Stack direction="row" spacing={2}>
              <TextField
                label={t('change_password_current')}
                variant="outlined"
                type="password"
                autoComplete="current-password"
                fullWidth
                {...register('currentPassword', {
                  required: true
                })}
              />
              <Box sx={{width: '100%'}} />
            </Stack>
            <Stack direction="row" spacing={2}>
              <TextField
                label={t('change_password_new')}
                variant="outlined"
                type="password"
                autoComplete="new-password"
                fullWidth
                {...register('newPassword', {
                  required: true
                })}
              />
              <TextField
                label={t('change_password_confirm')}
                variant="outlined"
                type="password"
                autoComplete="new-password"
                fullWidth
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
          <LoadingButton type="submit" variant="contained" disabled={!formState.isValid} loading={submitting} sx={buttonSx}>
            {success ? <CheckIcon /> : t('change_password_submit')}
          </LoadingButton>
        </Box>
      </form>

      <Snackbar onClose={() => setFeedback('')} message={feedback} />
    </>
  );
};

export default ChangePassword;
