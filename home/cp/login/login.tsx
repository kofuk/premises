import * as React from 'react';
import {FiAlertTriangle} from '@react-icons/all-files/fi/FiAlertTriangle';
import Modal from 'bootstrap/js/dist/modal';

import '../i18n';
import {t} from 'i18next';

type State = {
    isLoggingIn: boolean;
    userName: string;
    password: string;
    feedback: string;
    newPassword: string;
    newPasswordConfirm: string;
    canChangePassword: boolean;
    changePasswordFeedback: string;
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
        changePasswordFeedback: ''
    };

    componentDidMount = () => {
        document.title = t('title_login');
    };

    handleLogin = () => {
        this.setState({isLoggingIn: true});

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
        params.append('username', this.state.userName);
        params.append('password', this.state.password);
        params.append('new-password', this.state.newPassword);

        fetch('/api/settings/change-password', {
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
                                />
                                <label htmlFor="username">{t('username')}</label>
                            </div>
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
                                    />
                                    <label htmlFor="password">{t('password')}</label>
                                </div>
                            </div>
                            <div className="text-end">
                                <button
                                    type="submit"
                                    className="btn btn-primary bg-gradient"
                                    disabled={this.state.isLoggingIn || this.state.userName === '' || this.state.password === ''}
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
                                            <label htmlFor="newPassword">{t('password')}</label>
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
                                            <label htmlFor="password_confirm">{t('password_confirm')}</label>
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
