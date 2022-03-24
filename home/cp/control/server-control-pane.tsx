import * as React from 'react';
import {FaStop} from '@react-icons/all-files/fa/FaStop';

import '../i18n';
import {t} from 'i18next';

import ReconfigureMenu from './reconfigure-menu';
import Snapshot from './control-item/snapshot';
import SystemInfo from './control-item/system-info';
import WorldInfo from './control-item/world-info';

enum Modes {
    MainMenu,
    Reconfigure,
    Snapshot,
    SystemInfo,
    WorldInfo
}

type Prop = {
    showError: (message: string) => void;
};

type State = {
    mode: Modes;
};

export default class ServerControlPane extends React.Component<Prop, State> {
    state: State = {
        mode: Modes.MainMenu
    };

    handleBackToMenu = () => {
        this.setState({mode: Modes.MainMenu});
    };

    render = () => {
        const controlItems: React.ReactElement[] = [];

        if (this.state.mode === Modes.MainMenu) {
            controlItems.push(
                <div className="list-group" key="mainMenu">
                    <button
                        type="button"
                        className="list-group-item list-group-item-action"
                        onClick={() => {
                            this.setState({mode: Modes.WorldInfo});
                        }}
                    >
                        {t('menu_world_info')}
                    </button>
                    <button
                        type="button"
                        className="list-group-item list-group-item-action"
                        onClick={() => {
                            this.setState({mode: Modes.Reconfigure});
                        }}
                    >
                        {t('menu_reconfigure')}
                    </button>
                    <button
                        type="button"
                        className="list-group-item list-group-item-action"
                        onClick={() => {
                            this.setState({mode: Modes.Snapshot});
                        }}
                    >
                        {t('menu_snapshot')}
                    </button>
                    <button
                        type="button"
                        className="list-group-item list-group-item-action"
                        onClick={() => {
                            this.setState({mode: Modes.SystemInfo});
                        }}
                    >
                        {t('menu_system_info')}
                    </button>
                </div>
            );
        } else if (this.state.mode === Modes.Reconfigure) {
            controlItems.push(<ReconfigureMenu backToMenu={this.handleBackToMenu} showError={this.props.showError} key="reconfigure" />);
        } else if (this.state.mode === Modes.Snapshot) {
            controlItems.push(<Snapshot backToMenu={this.handleBackToMenu} showError={this.props.showError} key="snapshot" />);
        } else if (this.state.mode === Modes.SystemInfo) {
            controlItems.push(<SystemInfo backToMenu={this.handleBackToMenu} key="systemInfo" />);
        } else if (this.state.mode === Modes.WorldInfo) {
            controlItems.push(<WorldInfo backToMenu={this.handleBackToMenu} key="worldInfo" />);
        }

        return (
            <div className="my-5 card mx-auto">
                <div className="card-body">
                    <form>
                        {controlItems}
                        <div className="d-md-block mt-3 text-end">
                            <button
                                className="btn btn-danger bg-gradient"
                                type="button"
                                onClick={(e: React.MouseEvent) => {
                                    e.preventDefault();
                                    fetch('/control/api/stop', {method: 'post'});
                                }}
                            >
                                <FaStop /> {t('stop_server')}
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        );
    };
}
