import * as React from 'react';

import '../i18n';
import {t} from 'i18next';

import StatusBar from './statusbar';
import ServerControlPane from './server-control-pane';
import ServerConfigPane from './server-config-pane';
import Settings from './settings';
import {Navigate} from 'react-router-dom';
import {Helmet} from 'react-helmet-async';

type AppState = {
    isServerShutdown: boolean;
    isError: boolean;
    message: string;
    showNotificationToast: boolean;
    logout: boolean;
};

export default class App extends React.Component<{}, AppState> {
    useNotification: boolean = false;
    state: AppState = {
        isServerShutdown: true,
        isError: false,
        message: '',
        showNotificationToast: true,
        logout: false
    };

    private statusSource: EventSource | null = null;

    prevStatus: string = '';

    constructor(props: {}) {
        super(props);

        if (Notification.permission === 'granted') {
            this.state.showNotificationToast = false;
            this.useNotification = true;
        }
    }

    componentDidMount = () => {
        if (this.statusSource === null) {
            const eventSource = new EventSource('/api/status');
            eventSource.addEventListener('error', this.handleEventClose);
            eventSource.addEventListener('statuschanged', this.handleServerEvent);

            this.statusSource = eventSource;
        }

        fetch('/api/current-user')
            .then((resp) => resp.json())
            .then((resp) => {
                if (!resp['success']) {
                    this.setState({logout: true});
                }
            });
    };

    componentWillUnmount(): void {
        this.statusSource?.close();
        this.statusSource = null;
    }

    handleEventClose = (_: any) => {
        this.setState({isError: true, message: t('reconnecting')});
    };

    handleServerEvent = (ev: MessageEvent) => {
        const event = JSON.parse(ev.data);
        this.setState({isServerShutdown: event.shutdown, isError: event.hasError, message: event.status});

        //TODO: temporary implementation
        if (event.status !== this.prevStatus && this.prevStatus !== '' && (event.status === '実行中' || event.status === 'Running')) {
            new Notification(t('notification_title'), {body: t('notification_body')});
        }

        this.prevStatus = event.status;
    };

    showError = (message: string) => {
        this.setState({isError: true, message: message});
    };

    closeNotificationToast = () => {
        this.setState({showNotificationToast: false});
    };

    requestNotification = () => {
        Notification.requestPermission().then((result) => {
            this.useNotification = result === 'granted';
        });
        this.closeNotificationToast();
    };

    logout = () => {
        fetch('/logout', {
            method: 'POST'
        })
            .then((resp) => resp.json())
            .then((resp) => {
                if (resp['success']) {
                    this.setState({logout: true});
                }
            });
    };

    render = () => {
        const mainPane: React.ReactElement = this.state.isServerShutdown ? (
            <ServerConfigPane showError={this.showError} />
        ) : (
            <ServerControlPane showError={this.showError} />
        );
        return (
            <>
                {this.state.logout && <Navigate to="/" replace={true} />}
                <nav className="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
                    <div className="container-fluid">
                        <span className="navbar-brand">{t('app_name')}</span>
                        <div className="collapse navbar-collapse">
                            <div className="navbar-nav me-auto"></div>
                            <a
                                className="btn btn-link me-1"
                                data-bs-toggle="offcanvas"
                                href="#settingsPane"
                                role="button"
                                aria-controls="settingsPane"
                            >
                                {t('settings')}
                            </a>
                            <button onClick={this.logout} className="btn btn-primary bg-gradient">
                                {t('logout')}
                            </button>
                        </div>
                    </div>
                </nav>

                <div className="offcanvas offcanvas-start" tabIndex={-1} id="settingsPane" aria-labelledby="SettingsLabel">
                    <div className="offcanvas-header">
                        <h5 className="offcanvas-title" id="settingsLabel">
                            {t('settings')}
                        </h5>
                        <button type="button" className="btn-close text-reset" data-bs-dismiss="offcanvas" aria-label="Close"></button>
                    </div>
                    <div className="offcanvas-body">
                        <Settings />
                    </div>
                </div>

                <div className="container">
                    <StatusBar isError={this.state.isError} message={this.state.message} />
                    {mainPane}
                </div>

                <div className="toast-container position-absolute top-0 end-0 pe-1 pt-5">
                    <div className={`toast ${this.state.showNotificationToast ? 'show' : ''}`}>
                        <div className="toast-header">
                            <strong className="me-auto">{t('notification_toast_title')}</strong>
                            <button
                                type="button"
                                className="btn-close"
                                data-bs-dismiss="toast"
                                aria-label="Close"
                                onClick={this.closeNotificationToast}
                            ></button>
                        </div>
                        <div className="toast-body">
                            {t('notification_toast_description')}
                            <div className="text-end">
                                <button type="button" className="btn btn-primary btn-sm" onClick={this.requestNotification}>
                                    {t('notification_allow')}
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
                <Helmet>
                    <title>{t('app_name')}</title>
                </Helmet>
            </>
        );
    };
}
