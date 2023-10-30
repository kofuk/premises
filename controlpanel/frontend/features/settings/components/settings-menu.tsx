import React from 'react';
import {useNavigate} from 'react-router-dom';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import {Key as KeyIcon, Lock as LockIcon, PersonAdd as PersonAddIcon} from '@mui/icons-material';
import {Box, List, ListItemButton, ListItemIcon, ListItemText, ListSubheader} from '@mui/material';

import {usePasskeysSupported} from '@/utils/auth';

const SettingsMenu = () => {
  const [t] = useTranslation();

  const passkeysSupported = usePasskeysSupported();

  const navigate = useNavigate();

  return (
    <Box>
      <List subheader={<ListSubheader component="div">{t('settings_account_security')}</ListSubheader>}>
        <ListItemButton onClick={() => navigate('change-password')}>
          <ListItemIcon>
            <LockIcon />
          </ListItemIcon>
          <ListItemText primary={t('change_password_header')} />
        </ListItemButton>
        {passkeysSupported && (
          <ListItemButton onClick={() => navigate('passkeys')}>
            <ListItemIcon>
              <KeyIcon />
            </ListItemIcon>
            <ListItemText primary={t('passwordless_login')} />
          </ListItemButton>
        )}
      </List>
      <List subheader={<ListSubheader component="div">{t('settings_server_management')}</ListSubheader>}>
        <ListItemButton onClick={() => navigate('add-user')}>
          <ListItemIcon>
            <PersonAddIcon />
          </ListItemIcon>
          <ListItemText primary={t('add_user_header')} />
        </ListItemButton>
      </List>

      <Helmet>
        <title>
          {t('settings')} - {t('app_name')}
        </title>
      </Helmet>
    </Box>
  );
};

export default SettingsMenu;