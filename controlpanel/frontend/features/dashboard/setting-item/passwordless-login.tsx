import {useState, useEffect} from 'react';

import '@/i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '@/utils/base64url';
import {FaTrash} from '@react-icons/all-files/fa/FaTrash';

interface HardwareKey {
  id: string;
  name: string;
}

type Props = {
  updateFeedback: (message: string) => void;
};

export default (props: Props) => {
  const {updateFeedback} = props;

  const [keyName, setKeyName] = useState('');
  const [canContinue, setCanContinue] = useState(true);
  const [hardwareKeys, setHardwareKeys] = useState<HardwareKey[]>([]);

  const refreshHardwareKeys = () => {
    fetch('/api/hardwarekey')
      .then((resp) => resp.json())
      .then((resp) => {
        if (!resp['success']) {
          updateFeedback(resp['reason']);
          return;
        }
        setHardwareKeys(resp['data']);
      });
  };

  useEffect(() => {
    refreshHardwareKeys();
  }, []);

  const handleAddKey = () => {
    setCanContinue(false);

    fetch('/api/hardwarekey/begin', {
      method: 'post'
    })
      .then((resp) => resp.json())
      .then((resp) => {
        if (!resp['success']) {
          updateFeedback(resp['reason']);
          setCanContinue(true);
          return;
        }

        const options = resp.options;

        options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
        options.publicKey.user.id = decodeBuffer(options.publicKey.user.id);
        if (options.publicKey.excludeCredentials) {
          for (let i = 0; i < options.publicKey.excludeCredentials.length; i++) {
            options.publicKey.excludeCredentials[i].id = decodeBuffer(options.publicKey.excludeCredentials[i].id);
          }
        }

        return navigator.credentials.create(options);
      })
      .then((cred) => {
        if (!cred) {
          throw 'error';
        }

        let publicKeyCred = cred as PublicKeyCredential;
        let attestationObject = (publicKeyCred.response as AuthenticatorAttestationResponse).attestationObject;
        let clientDataJson = publicKeyCred.response.clientDataJSON;
        let rawId = publicKeyCred.rawId;

        return fetch('/api/hardwarekey/finish?name=' + encodeURI(keyName), {
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
        })
          .then((resp) => resp.json())
          .then((resp) => {
            if (!resp['success']) {
              updateFeedback(resp['reason']);
              setCanContinue(true);
              return;
            }
            setCanContinue(true);
            setKeyName('');
            refreshHardwareKeys();
          })
          .catch((_) => {
            updateFeedback(t('passwordless_login_error'));

            setCanContinue(true);
          });
      })
      .catch((_) => {
        updateFeedback(t('passwordless_login_error'));

        setCanContinue(true);
      });
  };

  const handleInputKeyName = (val: string) => {
    setKeyName(val);
  };

  const deleteKey = (id: string) => {
    fetch('/api/hardwarekey/' + id, {
      method: 'delete'
    }).then((resp) => {
      if (resp.status === 204) {
        refreshHardwareKeys();
      }
    });
  };

  return (
    <>
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
                      <FaTrash />
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
    </>
  );
};
