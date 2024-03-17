import React from 'react';
import {Route, Routes} from 'react-router-dom';

import {Box} from '@mui/material';

import AddUserPage from '../components/add-user';
import ChangePasswordPage from '../components/change-passowrd';
import SettingsMenuPage from '../components/settings-menu';

const SettingsRoutes = () => {
  return (
    <Box sx={{maxWidth: 1000, m: '0 auto', p: 2}}>
      <Routes>
        <Route element={<SettingsMenuPage />} path="/" />
        <Route element={<ChangePasswordPage />} path="/change-password" />
        <Route element={<AddUserPage />} path="/add-user" />
      </Routes>
    </Box>
  );
};

export default SettingsRoutes;
