import React from 'react';
import {Route, Routes} from 'react-router-dom';

import AddUserPage from '../components/add-user';
import ChangePasswordPage from '../components/change-passowrd';
import SettingsMenuPage from '../components/settings-menu';

const SettingsRoutes = () => {
  return (
    <Routes>
      <Route element={<SettingsMenuPage />} path="/" />
      <Route element={<ChangePasswordPage />} path="/change-password" />
      <Route element={<AddUserPage />} path="/add-user" />
    </Routes>
  );
};

export default SettingsRoutes;
