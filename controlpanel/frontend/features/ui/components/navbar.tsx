import React from 'react';
import {useNavigate} from 'react-router-dom';

import {useTranslation} from 'react-i18next';
import styled from 'styled-components';

import {Logout as LogoutIcon, Settings as SettingsIcon} from '@mui/icons-material';
import {IconButton, Toolbar, Tooltip, Typography} from '@mui/material';

import {useAuth} from '@/utils/auth';

const RoundedAppBar = styled.div`
  position: sticky;
  top: 10px;
  background-color: #262626;
  color: white;
  padding: 10px;
  margin: 10px;
  border-radius: 100px;
  box-shadow: 2px 5px 5px rgba(0, 0, 0, 0.3);
  z-index: 1;
`;

const NavBar = () => {
  const [t] = useTranslation();

  const navigate = useNavigate();
  const {logout} = useAuth();
  const handleLogout = () => {
    (async () => {
      try {
        await logout();
        navigate('/', {replace: true});
      } catch (err) {
        console.error(err);
      }
    })();
  };

  return (
    <RoundedAppBar>
      <Toolbar variant="dense">
        <Typography variant="h6" color="inherit" component="div" sx={{flexGrow: 1}}>
          {t('app_name')}
        </Typography>

        <Tooltip title={t('settings')}>
          <IconButton size="large" color="inherit" onClick={() => navigate('/settings')}>
            <SettingsIcon />
          </IconButton>
        </Tooltip>
        <Tooltip title={t('logout')}>
          <IconButton size="large" color="inherit" onClick={handleLogout}>
            <LogoutIcon />
          </IconButton>
        </Tooltip>
      </Toolbar>
    </RoundedAppBar>
  );
};

export default NavBar;
