import React, {useState} from 'react';

import {useSnackbar} from 'notistack';
import {Helmet} from 'react-helmet-async';
import {useForm} from 'react-hook-form';
import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';

import {Add as AddIcon, Delete as DeleteIcon} from '@mui/icons-material';
import {
  Alert,
  AlertTitle,
  CircularProgress,
  Collapse,
  Divider,
  IconButton,
  InputAdornment,
  List,
  ListItem,
  ListItemText,
  ListSubheader,
  TextField,
  Typography
} from '@mui/material';
import {Box} from '@mui/system';

import {APIError, getPasskeysRegistrationOptions, registerPasskeys, usePasskeys} from '@/api';
import {Loading} from '@/components';
import {decodeBuffer, encodeBuffer} from '@/utils/base64url';

const Passkeys = () => {
  const [t] = useTranslation();

  const [submitting, setSubmitting] = useState(false);

  const {register, handleSubmit, reset} = useForm();
  const {enqueueSnackbar} = useSnackbar();

  const {data: passkeys, isLoading, mutate, deleteKey} = usePasskeys();

  const handleAddKey = ({keyName}: any) => {
    (async () => {
      setSubmitting(true);

      try {
        const options = await getPasskeysRegistrationOptions();

        options.publicKey!.challenge = decodeBuffer(options.publicKey!.challenge as unknown as string);
        options.publicKey!.user.id = decodeBuffer(options.publicKey!.user.id as unknown as string);
        if (options.publicKey!.excludeCredentials) {
          for (let i = 0; i < options.publicKey!.excludeCredentials.length; i++) {
            options.publicKey!.excludeCredentials[i].id = decodeBuffer(options.publicKey!.excludeCredentials[i].id as unknown as string);
          }
        }

        const cred = await navigator.credentials.create(options);
        if (!cred) {
          throw new Error('Unable to create CredentialCreationResponse');
        }

        const publicKeyCred = cred as PublicKeyCredential;
        const attestationObject = (publicKeyCred.response as AuthenticatorAttestationResponse).attestationObject;
        const clientDataJson = publicKeyCred.response.clientDataJSON;
        const rawId = publicKeyCred.rawId;

        await registerPasskeys({
          name: keyName,
          credentialCreationResponse: {
            id: cred.id,
            rawId: encodeBuffer(rawId),
            type: publicKeyCred.type,
            response: {
              attestationObject: encodeBuffer(attestationObject),
              clientDataJSON: encodeBuffer(clientDataJson)
            }
          }
        });

        reset();
        mutate();
      } catch (err: unknown) {
        console.error(err);
        if (err instanceof APIError) {
          enqueueSnackbar(err.message, {variant: 'error'});
        } else {
          enqueueSnackbar(t('passwordless_login_error'), {variant: 'error'});
        }
      } finally {
        setSubmitting(false);
      }
    })();
  };

  const createPasskeyList = () => {
    return (
      <List subheader={<ListSubheader>{t('passwordless_login_existing_keys')}</ListSubheader>} sx={{mt: 5}}>
        <TransitionGroup>
          {passkeys!.map((passkey) => (
            <Collapse key={passkey.id}>
              <ListItem
                secondaryAction={
                  <IconButton aria-label="delete" onClick={() => deleteKey(passkey.id)}>
                    <DeleteIcon />
                  </IconButton>
                }
              >
                <ListItemText primary={passkey.name} />
              </ListItem>
              <Divider component="li" />
            </Collapse>
          ))}
        </TransitionGroup>
      </List>
    );
  };

  const createNoPasskeyMessage = () => {
    return (
      <Alert severity="info" sx={{mt: 5}}>
        <AlertTitle>{t('passwordless_login_no_keys')}</AlertTitle>
        {t('passwordless_login_no_keys_message')}
      </Alert>
    );
  };

  return (
    <>
      <Typography sx={{mt: 3}} variant="h4">
        {t('passwordless_login')}
      </Typography>

      <Box sx={{m: 2}}>
        <Typography variant="body1">{t('passwordless_login_description')}</Typography>

        <form onSubmit={handleSubmit(handleAddKey)}>
          <Box sx={{mt: 3, width: '30%'}}>
            <TextField
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton color="primary" disabled={submitting} type="submit">
                      {submitting ? <CircularProgress size={20} /> : <AddIcon />}
                    </IconButton>
                  </InputAdornment>
                )
              }}
              autoComplete="off"
              disabled={submitting}
              fullWidth
              label={t('passwordless_login_add')}
              type="text"
              variant="standard"
              {...register('keyName')}
            />
          </Box>
        </form>

        {isLoading ? <Loading compact /> : passkeys!.length > 0 ? createPasskeyList() : createNoPasskeyMessage()}
      </Box>

      <Helmet>
        <title>
          {t('passwordless_login')} - {t('app_name')}
        </title>
      </Helmet>
    </>
  );
};

export default Passkeys;
