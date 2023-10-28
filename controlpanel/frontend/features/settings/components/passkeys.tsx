import React, {useState} from 'react';

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
import Loading from '@/components/loading';
import Snackbar from '@/components/snackbar';
import {decodeBuffer, encodeBuffer} from '@/utils/base64url';

const Passkeys = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const [submitting, setSubmitting] = useState(false);

  const {register, handleSubmit, reset} = useForm();

  const {data: passkeys, isLoading, mutate, deleteKey} = usePasskeys();

  const handleAddKey = ({keyName}: any) => {
    (async () => {
      setSubmitting(true);

      try {
        const options = await getPasskeysRegistrationOptions();

        options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
        options.publicKey.user.id = decodeBuffer(options.publicKey.user.id);
        if (options.publicKey.excludeCredentials) {
          for (let i = 0; i < options.publicKey.excludeCredentials.length; i++) {
            options.publicKey.excludeCredentials[i].id = decodeBuffer(options.publicKey.excludeCredentials[i].id);
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
      } catch (err: Error) {
        console.error(err);
        if (err instanceof APIError) {
          setFeedback(err.message);
        } else {
          setFeedback(t('passwordless_login_error'));
        }
      } finally {
        setSubmitting(false);
      }
    })();
  };

  const handleInputKeyName = (val: string) => {
    setKeyName(val);
  };

  const createPasskeyList = () => {
    return (
      <List subheader={<ListSubheader>{t('passwordless_login_existing_keys')}</ListSubheader>} sx={{mt: 5}}>
        <TransitionGroup>
          {passkeys.map((passkey) => (
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
      <Typography variant="h4" sx={{mt: 3}}>
        {t('passwordless_login')}
      </Typography>

      <Box sx={{m: 2}}>
        <Typography variant="body1">{t('passwordless_login_description')}</Typography>

        <form onSubmit={handleSubmit(handleAddKey)}>
          <Box sx={{mt: 3, width: '30%'}}>
            <TextField
              variant="standard"
              type="text"
              label={t('passwordless_login_add')}
              autoComplete="off"
              onChange={(e) => handleInputKeyName(e.target.value)}
              disabled={submitting}
              InputProps={{
                endAdornment: (
                  <InputAdornment position="end">
                    <IconButton variant="outlined" color="primary" type="submit" disabled={submitting}>
                      {submitting ? <CircularProgress size={20} /> : <AddIcon />}
                    </IconButton>
                  </InputAdornment>
                )
              }}
              fullWidth
              {...register('keyName')}
            />
          </Box>
        </form>

        {isLoading ? <Loading compact /> : passkeys.length > 0 ? createPasskeyList() : createNoPasskeyMessage()}
      </Box>

      <Snackbar onClose={() => setFeedback('')} message={feedback} />

      <Helmet>
        <title>
          {t('passwordless_login')} - {t('app_name')}
        </title>
      </Helmet>
    </>
  );
};

export default Passkeys;
