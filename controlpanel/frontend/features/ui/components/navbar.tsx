import {t} from 'i18next';
import {Tooltip, Toolbar, IconButton, Typography} from '@mui/material';
import styled from 'styled-components';
import {Settings as SettingsIcon, Logout as LogoutIcon} from '@mui/icons-material';
import {useAuth} from '@/utils/auth';
import {useNavigate} from 'react-router-dom';

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

export default () => {
  const navigate = useNavigate();
  const {logout} = useAuth();
  const handleLogout = () => {
    logout().then(() => {
      navigate('/', {replace: true});
    });
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
