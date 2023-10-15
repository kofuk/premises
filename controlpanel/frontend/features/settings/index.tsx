import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import AddUser from './setting-item/add-user';
import ChangePassword from './setting-item/change-passowrd';
import PasswordlessLogin from './setting-item/passwordless-login';

import {passkeysSupported} from '@/utils/auth';

const SettingsPage = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const updateFeedback = (message: string) => {
    setFeedback(message);
  };

  const [passkeysAvailable, setPasskeysAvailable] = useState(false);
  useEffect(() => {
    (async () => {
      setPasskeysAvailable(await passkeysSupported());
    })();
  }, []);

  return (
    <>
      {feedback === '' ? null : <div className="alert alert-danger">{feedback}</div>}
      <div className="accordion" id="accordionSettings">
        {passkeysAvailable && (
          <div className="accordion-item">
            <h2 className="accordion-header" id="settings_heading_passwordlessLogin">
              <button
                className="accordion-button collapsed"
                type="button"
                data-bs-toggle="collapse"
                data-bs-target="#settings_passwordlessLogin"
                aria-expanded="false"
                aria-controls="settings_passwordlessLogin"
              >
                {t('passwordless_login')}
              </button>
            </h2>
            <div
              id="settings_passwordlessLogin"
              className="accordion-collapse collapse"
              aria-labelledby="settings_heading_passwordlessLogin"
              data-bs-parent="#accordionSettings"
            >
              <div className="accordion-body">
                <PasswordlessLogin updateFeedback={updateFeedback} />
              </div>
            </div>
          </div>
        )}
        <div className="accordion-item">
          <h2 className="accordion-header" id="settings_heading_changePassword">
            <button
              className="accordion-button collapsed"
              type="button"
              data-bs-toggle="collapse"
              data-bs-target="#settings_changePassword"
              aria-expanded="false"
              aria-controls="settings_changePassword"
            >
              {t('change_password_header')}
            </button>
          </h2>
          <div
            id="settings_changePassword"
            className="accordion-collapse collapse"
            aria-labelledby="settings_heading_changePassword"
            data-bs-parent="#accordionSettings"
          >
            <div className="accordion-body">
              <ChangePassword updateFeedback={updateFeedback} />
            </div>
          </div>
        </div>
        <div className="accordion-item">
          <h2 className="accordion-header" id="settings_heading_addUser">
            <button
              className="accordion-button collapsed"
              type="button"
              data-bs-toggle="collapse"
              data-bs-target="#settings_addUser"
              aria-expanded="false"
              aria-controls="settings_addUser"
            >
              {t('add_user_header')}
            </button>
          </h2>
          <div
            id="settings_addUser"
            className="accordion-collapse collapse"
            aria-labelledby="settings_heading_addUser"
            data-bs-parent="#accordionSettings"
          >
            <div className="accordion-body">
              <AddUser updateFeedback={updateFeedback} />
            </div>
          </div>
        </div>
      </div>
    </>
  );
};

export default SettingsPage;
