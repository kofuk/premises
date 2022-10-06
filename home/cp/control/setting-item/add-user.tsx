import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

type State = {
    canContinue: boolean;
    userName: string;
    password: string;
    passwordConfirm: string;
    success: boolean;
};

type Props = {
    updateFeedback: (message: string, negative: boolean) => void;
};

export default class AddUser extends React.Component<Props, State> {
    state = {
        canContinue: false,
        userName: '',
        password: '',
        passwordConfirm: '',
        success: false
    };

    handleAddUser = () => {
        this.setState({canContinue: false});

        const params = new URLSearchParams();
        params.append('username', this.state.userName);
        params.append('password', this.state.password);

        fetch('/api/users/add', {
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
                        canContinue: false,
                        userName: '',
                        password: '',
                        passwordConfirm: '',
                        success: true
                    });
                    return;
                }
                this.props.updateFeedback(resp['reason'], true);
            });
    };

    handleInputUserName = (val: string) => {
        this.setState({
            userName: val,
            canContinue: val !== '' && this.state.password.length >= 8 && this.state.password == this.state.passwordConfirm
        });
    };

    handleInputPassword = (val: string) => {
        this.setState({
            password: val,
            canContinue: this.state.userName !== '' && val.length >= 8 && val == this.state.passwordConfirm
        });
    };

    handleInputPasswordConfirm = (val: string) => {
        this.setState({
            passwordConfirm: val,
            canContinue: this.state.userName !== '' && this.state.password.length >= 8 && this.state.password == val
        });
    };

    render = () => {
        return (
            <form
                onSubmit={(e) => {
                    e.preventDefault();
                    this.handleAddUser();
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
                <div>
                    <div className="mb-3 form-floating">
                        <input
                            type="password"
                            id="password_confirm"
                            className="form-control"
                            placeholder="Confirm password"
                            onChange={(e) => this.handleInputPasswordConfirm(e.target.value)}
                            value={this.state.passwordConfirm}
                            required={true}
                        />
                        <label htmlFor="password_confirm">{t('password_confirm')}</label>
                    </div>
                </div>
                <div className="text-end">
                    {this.state.success ? <span className="text-success">✓ {t('add_user_success')}</span> : ''}
                    <button type="submit" className="btn btn-primary bg-gradient ms-3" disabled={!this.state.canContinue}>
                        {t('add_user_submit')}
                    </button>
                </div>
            </form>
        );
    };
}
