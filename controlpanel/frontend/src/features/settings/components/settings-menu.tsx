import {useNavigate} from 'react-router-dom';

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
              {t('settings.account_security')}
            </Typography>
            <List>
              <ListItemButton onClick={() => navigate('change-password')}>
                <ListItemIcon>
                  <LockIcon />
                </ListItemIcon>
                <ListItemText primary={t('settings.change_password')} />
              </ListItemButton>
            </List>
          </Paper>
        </Grid>

        <Grid item xs={6}>
          <Paper variant="outlined">
            <Typography sx={{m: 2}} variant="h5">
              {t('settings.server_manage')}
            </Typography>
            <List>
              <ListItemButton onClick={() => navigate('add-user')}>
                <ListItemIcon>
                  <PersonAddIcon />
                </ListItemIcon>
                <ListItemText primary={t('settings.add_user')} />
              </ListItemButton>
            </List>
          </Paper>
        </Grid>
      </Grid>

      <title>
        {t('settings.title')} - {t('app_name')}
      </title>
    </>
  );
};

export default SettingsMenu;
