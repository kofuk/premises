import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import Snackbar from '@/components/snackbar';

const AddUser = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const [canContinue, setCanContinue] = useState(false);
  const [userName, setUserName] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [success, setSuccess] = useState(false);

  const handleAddUser = () => {
    (async () => {
      setCanContinue(false);

      try {
        const params = new URLSearchParams();
        params.append('username', userName);
        params.append('password', password);

        const result = await fetch('/api/users/add', {
          method: 'post',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          body: params.toString()
        }).then((resp) => resp.json());
        if (result['success']) {
          setCanContinue(false);
          setUserName('');
          setPassword('');
          setPasswordConfirm('');
          setSuccess(true);
          return;
        }
        setFeedback(result['reason']);
      } catch (err) {
        console.error(err);
      }
    })();
  };

  const handleInputUserName = (val: string) => {
    setUserName(val);
    setCanContinue(val !== '' && password.length >= 8 && password == passwordConfirm);
  };

  const handleInputPassword = (val: string) => {
    setPassword(val);
    setCanContinue(userName !== '' && val.length >= 8 && val == passwordConfirm);
  };

  const handleInputPasswordConfirm = (val: string) => {
    setPasswordConfirm(val);
    setCanContinue(userName !== '' && password.length >= 8 && password == val);
  };

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        handleAddUser();
      }}
    >
      <h2>{t('add_user_header')}</h2>

      <div className="mb-3 form-floating">
        <input
          type="text"
          autoComplete="username"
          id="newUser_username"
          className="form-control"
          placeholder={t('username')}
          onChange={(e) => handleInputUserName(e.target.value)}
          value={userName}
          required={true}
        />
        <label htmlFor="newUser_username">{t('username')}</label>
      </div>
      <div>
        <div className="mb-3 form-floating">
          <input
            type="password"
            id="newUser_password"
            autoComplete="new-password"
            className="form-control"
            placeholder={t('password')}
            onChange={(e) => handleInputPassword(e.target.value)}
            value={password}
            required={true}
          />
          <label htmlFor="newUser_password">{t('password')}</label>
        </div>
      </div>
      <div>
        <div className="mb-3 form-floating">
          <input
            type="password"
            autoComplete="new-password"
            id="newUser_password_confirm"
            className="form-control"
            placeholder={t('password_confirm')}
            onChange={(e) => handleInputPasswordConfirm(e.target.value)}
            value={passwordConfirm}
            required={true}
          />
          <label htmlFor="newUser_password_confirm">{t('password_confirm')}</label>
        </div>
      </div>
      <div className="text-end">
        {success ? <span className="text-success">✓ {t('add_user_success')}</span> : ''}
        <button type="submit" className="btn btn-primary bg-gradient ms-3" disabled={!canContinue}>
          {t('add_user_submit')}
        </button>
      </div>

      <Snackbar onClose={() => setFeedback('')} message={feedback} />
    </form>
  );
};

export default AddUser;
