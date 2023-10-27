import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Delete as DeleteIcon} from '@mui/icons-material';

import Snackbar from '@/components/snackbar';
import {decodeBuffer, encodeBuffer} from '@/utils/base64url';

interface HardwareKey {
  id: string;
  name: string;
}

const Passkeys = () => {
  const [t] = useTranslation();

  const [feedback, setFeedback] = useState('');

  const [keyName, setKeyName] = useState('');
  const [canContinue, setCanContinue] = useState(true);
  const [hardwareKeys, setHardwareKeys] = useState<HardwareKey[]>([]);

  const refreshHardwareKeys = () => {
    (async () => {
      try {
        const result = await fetch('/api/hardwarekey').then((resp) => resp.json());
        if (!result['success']) {
          setFeedback(result['reason']);
          return;
        }
        setHardwareKeys(result['data']);
      } catch (err) {
        console.log(err);
      }
    })();
  };

  useEffect(() => {
    refreshHardwareKeys();
  }, []);

  const handleAddKey = () => {
    (async () => {
      setCanContinue(false);

      try {
        const beginResp = await fetch('/api/hardwarekey/begin', {
          method: 'post'
        }).then((resp) => resp.json());
        if (!beginResp['success']) {
          setFeedback(beginResp['reason']);
          return;
        }

        const options = beginResp.options;

        options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
        options.publicKey.user.id = decodeBuffer(options.publicKey.user.id);
        if (options.publicKey.excludeCredentials) {
          for (let i = 0; i < options.publicKey.excludeCredentials.length; i++) {
            options.publicKey.excludeCredentials[i].id = decodeBuffer(options.publicKey.excludeCredentials[i].id);
          }
        }

        const cred = await navigator.credentials.create(options);
        if (!cred) {
          throw 'error';
        }

        const publicKeyCred = cred as PublicKeyCredential;
        const attestationObject = (publicKeyCred.response as AuthenticatorAttestationResponse).attestationObject;
        const clientDataJson = publicKeyCred.response.clientDataJSON;
        const rawId = publicKeyCred.rawId;

        const finishResp = await fetch('/api/hardwarekey/finish?name=' + encodeURI(keyName), {
          method: 'post',
          body: JSON.stringify({
            id: cred.id,
            rawId: encodeBuffer(rawId),
            type: publicKeyCred.type,
            response: {
              attestationObject: encodeBuffer(attestationObject),
              clientDataJSON: encodeBuffer(clientDataJson)
            }
          })
        }).then((resp) => resp.json());

        if (!finishResp['success']) {
          setFeedback(finishResp['reason']);
          return;
        }
        setKeyName('');
        refreshHardwareKeys();
      } catch (err) {
        console.error(err);
        setFeedback(t('passwordless_login_error'));
      } finally {
        setCanContinue(true);
      }
    })();
  };

  const handleInputKeyName = (val: string) => {
    setKeyName(val);
  };

  const deleteKey = (id: string) => {
    (async () => {
      try {
        const resp = await fetch('/api/hardwarekey/' + id, {method: 'delete'});
        if (resp.status === 204) {
          refreshHardwareKeys();
        }
      } catch (err) {
        console.error(err);
      }
    })();
  };

  return (
    <>
      <h2>{t('passwordless_login')}</h2>
      <div className="mb-3">{t('passwordless_login_description')}</div>
      {hardwareKeys.length === 0 ? null : (
        <>
          <table className="table">
            <thead>
              <tr>
                <td></td>
                <td>{t('passwordless_login_key_name')}</td>
              </tr>
            </thead>
            <tbody>
              {hardwareKeys.map((e) => (
                <tr key={e.id}>
                  <td>
                    <button
                      type="button"
                      className="btn btn-danger bg-gradient"
                      onClick={(ev) => {
                        ev.preventDefault();
                        deleteKey(e.id);
                      }}
                    >
                      <DeleteIcon />
                    </button>
                  </td>
                  <td className="align-middle">{e.name}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
      <form
        onSubmit={(e) => {
          e.preventDefault();
          handleAddKey();
        }}
      >
        <div className="input-group">
          <input
            type="text"
            className="form-control"
            placeholder={t('passwordless_login_key_name')}
            onChange={(e) => handleInputKeyName(e.target.value)}
            value={keyName}
            disabled={!canContinue}
          />
          <button type="submit" className="btn btn-primary bg-gradient" disabled={!canContinue}>
            {t('passwordless_login_add')}
          </button>
        </div>
      </form>

      <Snackbar onClose={() => setFeedback('')} message={feedback} />
    </>
  );
};

export default Passkeys;
