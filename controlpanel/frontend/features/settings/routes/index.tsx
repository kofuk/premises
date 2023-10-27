import React from 'react';
import {Route, Routes} from 'react-router-dom';

const SettingsMenuPage = React.lazy(() => import('../components/settings-menu'));
const ChangePasswordPage = React.lazy(() => import('../components/change-passowrd'));
const PasskeysPage = React.lazy(() => import('../components/passkeys'));
const AddUserPage = React.lazy(() => import('../components/add-user'));

const SettingsRoutes = () => {
  return (
    <Routes>
      <Route path="/" element={<SettingsMenuPage />} />
      <Route path="/change-password" element={<ChangePasswordPage />} />
      <Route path="/passkeys" element={<PasskeysPage />} />
      <Route path="/add-user" element={<AddUserPage />} />
    </Routes>
  );
};

export default SettingsRoutes;
