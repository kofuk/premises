import * as React from 'react';

import '../i18n';
import {t} from 'i18next';

import StatusBar from './statusbar';
import ServerControlPane from './server-control-pane';
import ServerConfigPane from './server-config-pane';

type AppState = {
    isServerShutdown: boolean;
    isError: boolean;
    message: string;
    showNotificationToast: boolean;
};

export default class App extends React.Component<{}, AppState> {
    retryCount: number;
    socketUrl: string;
    useNotification: boolean = false;
    state: AppState = {
        isServerShutdown: true,
        isError: false,
        message: '',
        showNotificationToast: true
    };

    prevStatus: string = '';

    constructor(props: {}) {
        super(props);

        const proto: string = location.protocol == 'https:' ? 'wss://' : 'ws://';
        this.socketUrl = proto + location.host + '/api/status';
        this.retryCount = 0;

        if (Notification.permission === 'granted') {
            this.state.showNotificationToast = false;
            this.useNotification = true;
        }
    }

    componentDidMount = () => {
        this.wsWatch();

        document.title = t('app_name');
    };

    wsWatch = () => {
        this.setState({isError: false, message: t('connecting')});

        const ws: WebSocket = new WebSocket(this.socketUrl);
        ws.addEventListener('close', this.handleWsClose);
        ws.addEventListener('message', this.handleWsMessage);
    };

    handleWsOpen = () => {
        this.setState({isError: false, message: t('connected')});
        this.retryCount = 0;
    };

    handleWsClose = (e: any) => {
        if (e.wasClean) {
            return;
        }

        if (this.retryCount === 20) {
            this.setState({isError: true, message: t('disconnected')});
            return;
        } else {
            this.setState({isError: true, message: t('reconnecting')});
            this.retryCount++;
        }

        setTimeout(() => {
            const ws = new WebSocket(this.socketUrl);
            ws.addEventListener('open', this.handleWsOpen);
            ws.addEventListener('close', this.handleWsClose);
            ws.addEventListener('message', this.handleWsMessage);
        }, Math.random() * 5);
    };

    handleWsMessage = (ev: MessageEvent) => {
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

    render = () => {
        const mainPane: React.ReactElement = this.state.isServerShutdown ? (
            <ServerConfigPane showError={this.showError} />
        ) : (
            <ServerControlPane showError={this.showError} />
        );
        return (
            <div>
                <nav className="navbar navbar-expand-lg navbar-dark bg-dark mb-3">
                    <div className="container-fluid">
                        <span className="navbar-brand">{t('app_name')}</span>
                        <div className="collapse navbar-collapse">
                            <div className="navbar-nav me-auto"></div>
                            <a href="/logout" className="btn btn-primary bg-gradient">
                                {t('logout')}
                            </a>
                        </div>
                    </div>
                </nav>

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
            </div>
        );
    };
}
