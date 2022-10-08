import * as React from 'react';
import {FiAlertTriangle} from '@react-icons/all-files/fi/FiAlertTriangle';

import '../i18n';
import {t} from 'i18next';

type State = {
    canContinue: boolean;
    userName: string;
    password: string;
    passwordConfirm: string;
    feedback: string;
};

export default class SetupApp extends React.Component<{}, State> {
    state = {
        canContinue: false,
        userName: '',
        password: '',
        passwordConfirm: '',
        feedback: ''
    };

    componentDidMount = () => {
        document.title = t('title_setup');
    };

    handleContinue = () => {
        this.setState({canContinue: false});

        const params = new URLSearchParams();
        params.append('username', this.state.userName);
        params.append('password', this.state.password);

        fetch('/setup', {
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
                this.setState({canContinue: true, feedback: resp['reason']});
            });
    };

    handleInputUserName = (val: string) => {
        this.setState({userName: val});
    };

    handleInputPassword = (val: string) => {
        this.setState({
            password: val,
            canContinue: this.state.userName !== '' && this.state.password !== '' && val == this.state.passwordConfirm
        });
    };

    handleInputPasswordConfirm = (val: string) => {
        this.setState({
            passwordConfirm: val,
            canContinue: this.state.userName !== '' && this.state.password !== '' && val == this.state.password
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
                <div className="my-5 card mx-auto setup-card">
                    <div className="card-body">
                        <h1>{t('title_setup')}</h1>

                        <div className="mt-3">{t('setup_new_user_description')}</div>

                        <form
                            onSubmit={(e) => {
                                e.preventDefault();
                                this.handleContinue();
                            }}
                        >
                            <div className="mb-3 form-floating">
                                <input
                                    type="text"
                                    autoComplete="username"
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
                                        autoComplete="new-password"
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
                                        autoComplete="new-password"
                                        id="password_confirm"
                                        className="form-control"
                                        placeholder="Confirm password"
                                        onChange={(e) => this.handleInputPasswordConfirm(e.target.value)}
                                        required={true}
                                    />
                                    <label htmlFor="password_confirm">{t('password_confirm')}</label>
                                </div>
                            </div>
                            <div className="text-end">
                                <button type="submit" className="btn btn-primary bg-gradient" disabled={!this.state.canContinue}>
                                    {t('setup_continue')}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        );
    };
}
