import * as React from 'react';

import '../i18n';
import {t} from 'i18next';

import ChangePassword from './setting-item/change-passowrd';
import AddUser from './setting-item/add-user';

type State = {
    canChangePassword: boolean;
    currentPassword: string;
    newPassword: string;
    newPasswordConfirm: string;
    feedback: string;
    negativeFeedback: boolean;
    passwordSuccess: boolean;
};

export default class Settings extends React.Component<{}, State> {
    state = {
        canChangePassword: false,
        currentPassword: '',
        newPassword: '',
        newPasswordConfirm: '',
        feedback: '',
        negativeFeedback: false,
        passwordSuccess: false
    };

    updateFeedback = (message: string, negative: boolean) => {
        this.setState({
            feedback: message,
            negativeFeedback: negative
        });
    };

    render = () => {
        return (
            <>
                {this.state.feedback === '' ? null : <div className="alert alert-danger">{this.state.feedback}</div>}
                <div className="accordion" id="accordionSettings">
                    <div className="accordion-item">
                        <h2 className="accordion-header" id="settings_heading_changePassword">
                            <button
                                className="accordion-button"
                                type="button"
                                data-bs-toggle="collapse"
                                data-bs-target="#settings_changePassword"
                                aria-expanded="false"
                                aria-controls="settings_changePassword"
                            >
                                {t('change_password_header')}
                            </button>
                        </h2>
                        <div
                            id="settings_changePassword"
                            className="accordion-collapse collapse"
                            aria-labelledby="settings_heading_changePassword"
                            data-bs-parent="#accordionSettings"
                        >
                            <div className="accordion-body">
                                <ChangePassword updateFeedback={this.updateFeedback} />
                            </div>
                        </div>
                    </div>
                    <div className="accordion-item">
                        <h2 className="accordion-header" id="settings_heading_addUser">
                            <button
                                className="accordion-button collapsed"
                                type="button"
                                data-bs-toggle="collapse"
                                data-bs-target="#settings_addUser"
                                aria-expanded="false"
                                aria-controls="settings_addUser"
                            >
                                {t('add_user_header')}
                            </button>
                        </h2>
                        <div
                            id="settings_addUser"
                            className="accordion-collapse collapse"
                            aria-labelledby="settings_heading_addUser"
                            data-bs-parent="#accordionSettings"
                        >
                            <div className="accordion-body">
                                <AddUser updateFeedback={this.updateFeedback} />
                            </div>
                        </div>
                    </div>
                </div>
            </>
        );
    };
}
