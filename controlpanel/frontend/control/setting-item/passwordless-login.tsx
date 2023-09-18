import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '../../base64url';
import {FaTrash} from '@react-icons/all-files/fa/FaTrash';

interface HardwareKey {
    id: string;
    name: string;
}

type State = {
    keyName: string;
    canContinue: boolean;
    hardwareKeys: HardwareKey[];
};

type Props = {
    updateFeedback: (message: string, negative: boolean) => void;
};

export default class PasswordlessLogin extends React.Component<Props, State> {
    state = {
        keyName: '',
        canContinue: true,
        hardwareKeys: [] as HardwareKey[]
    };

    refreshHardwareKeys = () => {
        fetch('/api/hardwarekey')
            .then((resp) => resp.json())
            .then((resp) => {
                if (!resp['success']) {
                    this.props.updateFeedback(resp['reason'], true);
                    return;
                }
                this.setState({hardwareKeys: resp['data']});
            });
    };

    componentDidMount = () => {
        this.refreshHardwareKeys();
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
                        this.refreshHardwareKeys();
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

    deleteKey = (id: string) => {
        fetch('/api/hardwarekey/' + id, {
            method: 'delete'
        }).then((resp) => {
            if (resp.status === 204) {
                this.refreshHardwareKeys();
            }
        });
    };

    render = () => {
        return (
            <>
                <div className="mb-3">{t('passwordless_login_description')}</div>
                {this.state.hardwareKeys.length === 0 ? null : (
                    <>
                        <table className="table">
                            <thead>
                                <tr>
                                    <td></td>
                                    <td>{t('passwordless_login_key_name')}</td>
                                </tr>
                            </thead>
                            <tbody>
                                {this.state.hardwareKeys.map((e) => (
                                    <tr key={e.id}>
                                        <td>
                                            <button
                                                type="button"
                                                className="btn btn-danger bg-gradient"
                                                onClick={(ev) => {
                                                    ev.preventDefault();
                                                    this.deleteKey(e.id);
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
                        this.handleAddKey();
                    }}
                >
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
