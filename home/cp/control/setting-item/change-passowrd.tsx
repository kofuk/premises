import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

type State = {
    canChangePassword: boolean;
    currentPassword: string;
    newPassword: string;
    newPasswordConfirm: string;
    passwordSuccess: boolean;
};

type Props = {
    updateFeedback: (message: string, negative: boolean) => void;
};

export default class ChangePassword extends React.Component<Props, State> {
    state = {
        canChangePassword: false,
        currentPassword: '',
        newPassword: '',
        newPasswordConfirm: '',
        passwordSuccess: false
    };

    canActivateChangePasswordButton = (curPassword: string, newPassword: string, newPasswordConfirm: string): boolean => {
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

    handleInputCurrentPassword = (val: string) => {
        this.setState({
            currentPassword: val,
            canChangePassword: this.canActivateChangePasswordButton(val, this.state.newPassword, this.state.newPasswordConfirm),
            passwordSuccess: false
        });
    };

    handleInputNewPassword = (val: string) => {
        this.setState({
            newPassword: val,
            canChangePassword: this.canActivateChangePasswordButton(this.state.currentPassword, val, this.state.newPasswordConfirm),
            passwordSuccess: false
        });
    };

    handleInputPasswordConfirm = (val: string) => {
        this.setState({
            newPasswordConfirm: val,
            canChangePassword: this.canActivateChangePasswordButton(this.state.currentPassword, this.state.newPassword, val),
            passwordSuccess: false
        });
    };

    handleChangePassword = () => {
        this.setState({canChangePassword: false});

        const params = new URLSearchParams();
        params.append('password', this.state.currentPassword);
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
                    this.setState({
                        currentPassword: '',
                        newPassword: '',
                        newPasswordConfirm: '',
                        canChangePassword: true,
                        passwordSuccess: true
                    });
                    this.props.updateFeedback('', false);
                } else {
                    this.setState({
                        canChangePassword: true,
                        passwordSuccess: false
                    });
                    this.props.updateFeedback(resp['reason'], true);
                }
            });
    };

    render = () => {
        return (
            <form
                onSubmit={(e) => {
                    e.preventDefault();
                    this.handleChangePassword();
                }}
            >
                <div className="mb-3 form-floating">
                    <input
                        type="password"
                        id="username"
                        className="form-control"
                        placeholder="User"
                        onChange={(e) => this.handleInputCurrentPassword(e.target.value)}
                        value={this.state.currentPassword}
                        required={true}
                    />
                    <label htmlFor="username">{t('change_password_current')}</label>
                </div>
                <div>
                    <div className="mb-3 form-floating">
                        <input
                            type="password"
                            id="password"
                            className="form-control"
                            placeholder="Password"
                            onChange={(e) => this.handleInputNewPassword(e.target.value)}
                            value={this.state.newPassword}
                            required={true}
                        />
                        <label htmlFor="password">{t('change_password_new')}</label>
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
                <div className="text-end">
                    {this.state.passwordSuccess ? <span className="text-success">✓ {t('change_password_success')}</span> : ''}
                    <button type="submit" className="btn btn-primary bg-gradient ms-3" disabled={!this.state.canChangePassword}>
                        {t('change_password_submit')}
                    </button>
                </div>
            </form>
        );
    };
}
