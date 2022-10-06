import * as React from 'react';
import {FiAlertTriangle} from '@react-icons/all-files/fi/FiAlertTriangle';
import Modal from 'bootstrap/js/dist/modal';

import '../i18n';
import {t} from 'i18next';
import {encodeBuffer, decodeBuffer} from '../base64url';

type State = {
    isLoggingIn: boolean;
    userName: string;
    password: string;
    feedback: string;
    newPassword: string;
    newPasswordConfirm: string;
    canChangePassword: boolean;
    changePasswordFeedback: string;
    useHardwareKey: boolean;
};

export default class LoginApp extends React.Component<{}, State> {
    state = {
        isLoggingIn: false,
        userName: '',
        password: '',
        feedback: '',
        newPassword: '',
        newPasswordConfirm: '',
        canChangePassword: false,
        changePasswordFeedback: '',
        useHardwareKey: false
    };

    componentDidMount = () => {
        document.title = t('title_login');
    };

    handleNormalLogin = () => {
        const params = new URLSearchParams();
        params.append('username', this.state.userName);
        params.append('password', this.state.password);

        fetch('/login', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    if (resp['needsChangePassword']) {
                        new Modal('#changePassword', {}).show();
                    } else {
                        location.reload();
                    }
                    return;
                }
                this.setState({isLoggingIn: false, feedback: resp['reason']});
            });
    };

    handlePasswordlessLogin = () => {
        const params = new URLSearchParams();
        params.append('username', this.state.userName);

        fetch('/login/hardwarekey/begin', {
            method: 'post',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded'
            },
            body: params.toString()
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (!resp['success']) {
                    this.setState({isLoggingIn: false, feedback: resp['reason']});
                    return;
                }

                const options = resp.options;

                options.publicKey.challenge = decodeBuffer(options.publicKey.challenge);
                if (options.publicKey.allowCredentials) {
                    for (let i = 0; i < options.publicKey.allowCredentials.length; i++) {
                        options.publicKey.allowCredentials[i].id = decodeBuffer(options.publicKey.allowCredentials[i].id);
                    }
                }

                return navigator.credentials.get(options);
            })
            .then((assertion) => {
                let publicKeyCred = assertion as PublicKeyCredential;
                let authenticatorResp = publicKeyCred.response as AuthenticatorAssertionResponse;
                let authData = authenticatorResp.authenticatorData;
                let clientDataJson = publicKeyCred.response.clientDataJSON;
                let rawId = publicKeyCred.rawId;
                let sig = authenticatorResp.signature;
                let userHandle = authenticatorResp.userHandle!!;

                fetch('/login/hardwarekey/finish', {
                    method: 'post',
                    body: JSON.stringify({
                        id: assertion!!.id,
                        rawId: encodeBuffer(rawId),
                        type: publicKeyCred.type,
                        response: {
                            authenticatorData: encodeBuffer(authData),
                            clientDataJSON: encodeBuffer(clientDataJson),
                            signature: encodeBuffer(sig),
                            userHandle: encodeBuffer(userHandle)
                        }
                    })
                })
                    .then((resp) => resp.json())
                    .then((resp) => {
                        if (!resp['success']) {
                            this.setState({isLoggingIn: false, feedback: resp['reason']});
                            return;
                        }
                        location.reload();
                    })
                    .catch((e) => {
                        this.setState({isLoggingIn: false, feedback: t('passwordless_login_error')});
                    });
            })
            .catch((e) => {
                this.setState({isLoggingIn: false, feedback: t('passwordless_login_error')});
            });
    };

    handleLogin = () => {
        this.setState({isLoggingIn: true});

        if (this.state.useHardwareKey) {
            this.handlePasswordlessLogin();
        } else {
            this.handleNormalLogin();
        }
    };

    handleInputUserName = (val: string) => {
        this.setState({userName: val});
    };

    handleInputPassword = (val: string) => {
        this.setState({password: val});
    };

    handleInputNewPassword = (val: string) => {
        this.setState({
            newPassword: val,
            canChangePassword: val.length >= 8 && val === this.state.newPasswordConfirm
        });
    };

    handleInputPasswordConfirm = (val: string) => {
        this.setState({
            newPasswordConfirm: val,
            canChangePassword: this.state.newPassword.length >= 8 && this.state.newPassword === val
        });
    };

    handleChangePassword = () => {
        this.setState({canChangePassword: false});

        const params = new URLSearchParams();
        params.append('password', this.state.password);
        params.append('new-password', this.state.newPassword);

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
                    location.reload();
                    return;
                }
                this.setState({
                    changePasswordFeedback: resp['reason']
                });
            });
    };

    render = () => {
        return (
            <div className="container">
                {this.state.feedback !== '' ? (
                    <div className="m-3 alert alert-danger d-flex align-items-center" role="alert">
                        <FiAlertTriangle size={25} />
                        {this.state.feedback}
                    </div>
                ) : null}
                <div className="my-5 card mx-auto login-card">
                    <div className="card-body">
                        <h1>{t('title_login')}</h1>
                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                this.handleLogin();
                            }}
                        >
                            <div className="mb-3 form-floating">
                                <input
                                    type="text"
                                    id="username"
                                    className="form-control"
                                    placeholder="User"
                                    onChange={(e) => this.handleInputUserName(e.target.value)}
                                    value={this.state.userName}
                                    required={true}
                                    disabled={this.state.isLoggingIn}
                                />
                                <label htmlFor="username">{t('username')}</label>
                            </div>
                            {this.state.useHardwareKey ? null : (
                                <div>
                                    <div className="mb-3 form-floating">
                                        <input
                                            type="password"
                                            id="password"
                                            className="form-control"
                                            placeholder="Password"
                                            onChange={(e) => this.handleInputPassword(e.target.value)}
                                            value={this.state.password}
                                            required={true}
                                            disabled={this.state.isLoggingIn}
                                        />
                                        <label htmlFor="password">{t('password')}</label>
                                    </div>
                                </div>
                            )}
                            <div className="text-end">
                                {this.state.isLoggingIn ? null : (
                                    <button
                                        type="button"
                                        className="btn btn-link me-1"
                                        onClick={(e) => {
                                            e.preventDefault();
                                            this.setState({useHardwareKey: !this.state.useHardwareKey});
                                        }}
                                    >
                                        {this.state.useHardwareKey ? t('login_dont_use_hardware_key') : t('login_use_hardware_key')}
                                    </button>
                                )}
                                <button
                                    type="submit"
                                    className="btn btn-primary bg-gradient"
                                    disabled={
                                        this.state.isLoggingIn ||
                                        this.state.userName === '' ||
                                        (!this.state.useHardwareKey && this.state.password === '')
                                    }
                                >
                                    {this.state.isLoggingIn ? (
                                        <>
                                            <div className="spinner-border spinner-border-sm me-1" role="status"></div>
                                            {t('logging_in')}
                                        </>
                                    ) : (
                                        t('login')
                                    )}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
                <div
                    className="modal fade"
                    id="changePassword"
                    data-bs-backdrop="static"
                    data-bs-keyboard="false"
                    tabIndex={-1}
                    aria-labelledby="changePasswordLabel"
                    aria-hidden="true"
                >
                    <div className="modal-dialog">
                        <div className="modal-content">
                            <div className="modal-header">
                                <h5 className="modal-title" id="changePasswordLabel">
                                    {t('set_password_title')}
                                </h5>
                            </div>
                            {this.state.changePasswordFeedback === '' ? null : (
                                <div className="alert alert-danger m-3">{this.state.changePasswordFeedback}</div>
                            )}
                            <form
                                onSubmit={(e) => {
                                    e.preventDefault();
                                    this.handleChangePassword();
                                }}
                            >
                                <div className="modal-body">
                                    <div>
                                        <div className="mb-3 form-floating">
                                            <input
                                                type="password"
                                                id="newPassword"
                                                className="form-control"
                                                placeholder="Password"
                                                onChange={(e) => this.handleInputNewPassword(e.target.value)}
                                                value={this.state.newPassword}
                                                required={true}
                                            />
                                            <label htmlFor="newPassword">{t('change_password_new')}</label>
                                        </div>
                                    </div>
                                    <div>
                                        <div className="mb-3 form-floating">
                                            <input
                                                type="password"
                                                id="password_confirm"
                                                className="form-control"
                                                placeholder="Confirm password"
                                                onChange={(e) => this.handleInputPasswordConfirm(e.target.value)}
                                                value={this.state.newPasswordConfirm}
                                                required={true}
                                            />
                                            <label htmlFor="password_confirm">{t('change_password_confirm')}</label>
                                        </div>
                                    </div>
                                </div>
                                <div className="modal-footer">
                                    <button type="submit" className="btn btn-primary" disabled={!this.state.canChangePassword}>
                                        {t('set_password_submit')}
                                    </button>
                                </div>
                            </form>
                        </div>
                    </div>
                </div>
            </div>
        );
    };
}
