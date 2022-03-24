import * as React from 'react';

import '../i18n';
import {t} from 'i18next';

export default class LoginApp extends React.Component {
    componentDidMount = () => {
        document.title = t('title_login');
    };

    render = () => {
        return (
            <div className="container">
                <div className="my-5 card mx-auto login-card">
                    <div className="card-body">
                        <h1>{t('title_login')}</h1>
                        <form action="/" method="post">
                            <div className="mb-3 form-floating">
                                <input type="text" name="username" id="username" className="form-control" placeholder="User" />
                                <label htmlFor="username">{t('username')}</label>
                            </div>
                            <div>
                                <div className="mb-3 form-floating">
                                    <input type="password" name="password" id="password" className="form-control" placeholder="Password" />
                                    <label htmlFor="password">{t('password')}</label>
                                </div>
                            </div>
                            <div className="text-right">
                                <button type="submit" className="btn btn-primary bg-gradient">
                                    {t('login')}
                                </button>
                            </div>
                        </form>
                    </div>
                </div>
            </div>
        );
    };
}
