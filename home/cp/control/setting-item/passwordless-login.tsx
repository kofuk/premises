import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '../../base64url';

type State = {
    keyName: string;
    canContinue: boolean;
};

type Props = {
    updateFeedback: (message: string, negative: boolean) => void;
};

export default class PasswordlessLogin extends React.Component<Props, State> {
    state = {
        keyName: '',
        canContinue: true
    };

    handleAddKey = () => {
        this.setState({canContinue: false});

        fetch('/api/hardwarekey/begin', {
            method: 'post'
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (!resp['success']) {
                    this.props.updateFeedback(resp['reason'], true);
                    this.setState({canContinue: true});
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

                return fetch('/api/hardwarekey/finish?name=' + encodeURI(this.state.keyName), {
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
                            this.props.updateFeedback(resp['reason'], true);
                            this.setState({canContinue: true});
                            return;
                        }
                        this.setState({canContinue: true, keyName: ''});
                    })
                    .catch((e) => {
                        this.props.updateFeedback(t('passwordless_login_error'), true);

                        this.setState({canContinue: true});
                    });
            })
            .catch((e) => {
                this.props.updateFeedback(t('passwordless_login_error'), true);

                this.setState({canContinue: true});
            });
    };

    handleInputKeyName = (val: string) => {
        this.setState({keyName: val});
    };

    render = () => {
        return (
            <>
                <form
                    onSubmit={(e) => {
                        e.preventDefault();
                        this.handleAddKey();
                    }}
                >
                    <div className="mb-3">{t('passwordless_login_description')}</div>
                    <div className="input-group">
                        <input
                            type="text"
                            className="form-control"
                            placeholder={t('passwordless_login_key_name')}
                            onChange={(e) => this.handleInputKeyName(e.target.value)}
                            value={this.state.keyName}
                            disabled={!this.state.canContinue}
                        />
                        <button type="submit" className="btn btn-primary bg-gradient" disabled={!this.state.canContinue}>
                            {t('passwordless_login_add')}
                        </button>
                    </div>
                </form>
            </>
        );
    };
}
