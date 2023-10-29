import React from 'react';
import {Route, Routes} from 'react-router-dom';

const SettingsMenuPage = React.lazy(() => import('../components/settings-menu'));
const ChangePasswordPage = React.lazy(() => import('../components/change-passowrd'));
const PasskeysPage = React.lazy(() => import('../components/passkeys'));
const AddUserPage = React.lazy(() => import('../components/add-user'));

const SettingsRoutes = () => {
  return (
    <Routes>
      <Route element={<SettingsMenuPage />} path="/" />
      <Route element={<ChangePasswordPage />} path="/change-password" />
      <Route element={<PasskeysPage />} path="/passkeys" />
      <Route element={<AddUserPage />} path="/add-user" />
    </Routes>
  );
};

export default SettingsRoutes;
