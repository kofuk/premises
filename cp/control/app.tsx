import * as React from 'react';

import StatusBar from './statusbar';
import ServerControlPane from './server-control-pane';
import ServerConfigPane from './server-config-pane';

type AppState = {
    isServerShutdown: boolean;
    isError: boolean;
    message: string;
};

export default class App extends React.Component<{}, AppState> {
    retryCount: number;
    socketUrl: string;
    state: AppState = {
        isServerShutdown: true,
        isError: false,
        message: ''
    };

    constructor(props: {}) {
        super(props);

        const proto: string = location.protocol == 'https:' ? 'wss://' : 'ws://';
        this.socketUrl = proto + location.host + '/control/api/status';
        this.retryCount = 0;
    }

    componentDidMount = () => {
        this.wsWatch();
    };

    wsWatch = () => {
        const ws: WebSocket = new WebSocket(this.socketUrl);
        ws.addEventListener('close', this.handleWsClose);
        ws.addEventListener('message', this.handleWsMessage);
    };

    handleWsOpen = () => {
        this.setState({isError: false, message: 'Connected.'});
        this.retryCount = 0;
    };

    handleWsClose = () => {
        if (this.retryCount === 20) {
            this.setState({isError: true, message: 'Connection has lost; Please reload the page.'});
            return;
        } else {
            this.setState({isError: true, message: 'Connection has lost; Reconnecting...'});
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
    };

    showError = (message: string) => {
        this.setState({isError: true, message: message});
    };

    render = () => {
        const mainPane: React.ReactElement = this.state.isServerShutdown ? (
            <ServerConfigPane showError={this.showError} />
        ) : (
            <ServerControlPane showError={this.showError} />
        );
        return (
            <div className="container">
                <StatusBar isError={this.state.isError} message={this.state.message} />
                {mainPane}
            </div>
        );
    };
}
