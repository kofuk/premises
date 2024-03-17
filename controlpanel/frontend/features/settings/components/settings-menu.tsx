import React from 'react';
import {useNavigate} from 'react-router-dom';

import {Helmet} from 'react-helmet-async';
import {useTranslation} from 'react-i18next';

import {Lock as LockIcon, PersonAdd as PersonAddIcon} from '@mui/icons-material';
import {Grid, List, ListItemButton, ListItemIcon, ListItemText, Paper, Typography} from '@mui/material';

const SettingsMenu = () => {
  const [t] = useTranslation();

  const navigate = useNavigate();

  return (
    <>
      <Grid container spacing={1}>
        <Grid item xs={6}>
          <Paper variant="outlined">
            <Typography sx={{m: 2}} variant="h5">
              {t('settings_account_security')}
            </Typography>
            <List>
              <ListItemButton onClick={() => navigate('change-password')}>
                <ListItemIcon>
                  <LockIcon />
                </ListItemIcon>
                <ListItemText primary={t('change_password_header')} />
              </ListItemButton>
            </List>
          </Paper>
        </Grid>

        <Grid item xs={6}>
          <Paper variant="outlined">
            <Typography sx={{m: 2}} variant="h5">
              {t('settings_server_management')}
            </Typography>
            <List>
              <ListItemButton onClick={() => navigate('add-user')}>
                <ListItemIcon>
                  <PersonAddIcon />
                </ListItemIcon>
                <ListItemText primary={t('add_user_header')} />
              </ListItemButton>
            </List>
          </Paper>
        </Grid>
      </Grid>

      <Helmet>
        <title>
          {t('settings')} - {t('app_name')}
        </title>
      </Helmet>
    </>
  );
};

export default SettingsMenu;
