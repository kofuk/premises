import React from 'react';

import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';

import {Button, Dialog, DialogActions, DialogContent, DialogTitle, Stack, TextField} from '@mui/material';

type Props = {
  add: (key: string, value: string) => void;
  open: boolean;
  onClose: () => void;
};

const ServerPropsDialog = ({add, open, onClose}: Props) => {
  const [t] = useTranslation();

  const {register, handleSubmit, formState, reset} = useForm();

  const handleSave = ({key, value}: any) => {
    add(key, value);
    onClose();
    reset();
  };

  return (
    <Dialog onClose={onClose} open={open}>
      <form onSubmit={handleSubmit(handleSave)}>
        <DialogTitle>{t('additional_server_properties_add')}</DialogTitle>
        <DialogContent>
          <Stack spacing={1} sx={{mt: 1, minWidth: 400}}>
            <TextField fullWidth label={t('server_props_key')} variant="outlined" {...register('key', {required: true})} />
            <TextField fullWidth label={t('server_props_value')} variant="outlined" {...register('value')} />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button autoFocus onClick={onClose}>
            {t('cancel')}
          </Button>
          <Button autoFocus disabled={!formState.isValid} type="submit">
            {t('add')}
          </Button>
        </DialogActions>
      </form>
    </Dialog>
  );
};

export default ServerPropsDialog;