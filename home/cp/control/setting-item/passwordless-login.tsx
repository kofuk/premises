import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

type State = {
    keyName: string;
    canContinue: boolean;
};

type Props = {
    updateFeedback: (message: string, negative: boolean) => void;
};

const decodeBuffer = (value: string): Uint8Array => {
    return Uint8Array.from(atob(value), (c) => c.charCodeAt(0));
};

const encodeBuffer = (value: ArrayBuffer): string => {
    return btoa(String.fromCharCode.apply(null, new Uint8Array(value) as unknown as number[]))
        .replace(/\+/g, '-')
        .replace(/\//g, '_')
        .replace(/=+/g, '');
};

export default class PasswordlessLogin extends React.Component<Props, State> {
    state = {
        keyName: '',
        canContinue: true
    };

    handleAddKey = () => {
        this.setState({canContinue: false});

        fetch('/api/settings/hardwarekey/begin', {
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

                return fetch('/api/settings/hardwarekey/finish?name=' + encodeURI(this.state.keyName), {
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
                        this.setState({canContinue: true});
                    })
                    .catch((e) => {
                        this.props.updateFeedback('Operation was timed out or not allowed', true);

                        this.setState({canContinue: true});
                    });
            })
            .catch((e) => {
                console.error(e);

                this.props.updateFeedback('Operation was timed out or not allowed', true);

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
                    <div className="mb-3">You can login to Control Panel without password, using hardware security key.</div>
                    <div className="input-group">
                        <input
                            type="text"
                            className="form-control"
                            placeholder="Key name"
                            onChange={(e) => this.handleInputKeyName(e.target.value)}
                            value={this.state.keyName}
                        />
                        <button type="submit" className="btn btn-primary bg-gradient" disabled={!this.state.canContinue}>
                            Add
                        </button>
                    </div>
                </form>
            </>
        );
    };
}
