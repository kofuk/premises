import {useState} from 'react';

import '../../i18n';
import {t} from 'i18next';

type Props = {
  updateFeedback: (message: string) => void;
};

export default (props: Props) => {
  const {updateFeedback} = props;
  const [canChangePassword, setCanChangePassword] = useState(false);
  const [currentPassword, setCurrentPassword] = useState('');
  const [newPassword, setNewPassword] = useState('');
  const [newPasswordConfirm, setNewPasswordConfirm] = useState('');
  const [passwordSuccess, setPasswordSuccess] = useState(false);

  const canActivateChangePasswordButton = (curPassword: string, newPassword: string, newPasswordConfirm: string): boolean => {
    if (curPassword.length === 0) {
      return false;
    }
    if (newPassword.length < 8) {
      return false;
    }
    if (newPassword !== newPasswordConfirm) {
      return false;
    }
    return true;
  };

  const handleInputCurrentPassword = (val: string) => {
    setCurrentPassword(val);
    setCanChangePassword(canActivateChangePasswordButton(val, newPassword, newPasswordConfirm));
    setPasswordSuccess(false);
  };

  const handleInputNewPassword = (val: string) => {
    setNewPassword(val);
    setCanChangePassword(canActivateChangePasswordButton(currentPassword, val, newPasswordConfirm));
    setPasswordSuccess(false);
  };

  const handleInputPasswordConfirm = (val: string) => {
    setNewPasswordConfirm(val);
    setCanChangePassword(canActivateChangePasswordButton(currentPassword, newPassword, val));
    setPasswordSuccess(false);
  };

  const handleChangePassword = () => {
    setCanChangePassword(false);

    const params = new URLSearchParams();
    params.append('password', currentPassword);
    params.append('new-password', newPassword);

    fetch('/api/users/change-password', {
      method: 'post',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: params.toString()
    })
      .then((resp) => resp.json())
      .then((resp) => {
        if (resp['success']) {
          setCurrentPassword('');
          setNewPassword('');
          setNewPasswordConfirm('');
          setCanChangePassword(true);
          setPasswordSuccess(true);
          updateFeedback('');
        } else {
          setCanChangePassword(true);
          setPasswordSuccess(false);
          updateFeedback(resp['reason']);
        }
      });
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        handleChangePassword();
      }}
    >
      <div className="mb-3 form-floating">
        <input
          type="password"
          autoComplete="current-password"
          id="changePassword_username"
          className="form-control"
          placeholder={t('change_password_current')}
          onChange={(e) => handleInputCurrentPassword(e.target.value)}
          value={currentPassword}
          required={true}
        />
        <label htmlFor="changePassword_username">{t('change_password_current')}</label>
      </div>
      <div>
        <div className="mb-3 form-floating">
          <input
            type="password"
            autoComplete="new-password"
            id="changePassword_password"
            className="form-control"
            placeholder={t('change_password_new')}
            onChange={(e) => handleInputNewPassword(e.target.value)}
            value={newPassword}
            required={true}
          />
          <label htmlFor="changePassword_password">{t('change_password_new')}</label>
        </div>
      </div>
      <div>
        <div className="mb-3 form-floating">
          <input
            type="password"
            autoComplete="new-password"
            id="changePassword_password_confirm"
            className="form-control"
            placeholder={t('change_password_confirm')}
            onChange={(e) => handleInputPasswordConfirm(e.target.value)}
            value={newPasswordConfirm}
            required={true}
          />
          <label htmlFor="changePassword_password_confirm">{t('change_password_confirm')}</label>
        </div>
      </div>
      <div className="text-end">
        {passwordSuccess ? <span className="text-success">✓ {t('change_password_success')}</span> : ''}
        <button type="submit" className="btn btn-primary bg-gradient ms-3" disabled={!canChangePassword}>
          {t('change_password_submit')}
        </button>
      </div>
    </form>
  );
};
