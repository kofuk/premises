import * as React from 'react';
import {FaStop} from '@react-icons/all-files/fa/FaStop';

import ReconfigureMenu from './reconfigure-menu';
import Snapshot from './control-item/snapshot';
import SystemInfo from './control-item/system-info';

enum Modes {
    MainMenu,
    Reconfigure,
    Snapshot,
    SystemInfo
};

type Prop = {
    showError: (message: string) => void
}

type State = {
    mode: Modes
};

export default class ServerControlPane extends React.Component<Prop, State> {
    state: State = {
        mode: Modes.MainMenu
    };

    handleBackToMenu = () => {
        this.setState({mode: Modes.MainMenu});
    };

    render = () => {
        const controlItems: React.ReactElement[] = []

        if (this.state.mode === Modes.MainMenu) {
            controlItems.push(
                <div className="list-group" key="mainMenu">
                    <button type="button" className="list-group-item list-group-item-action"
                            onClick={() => {this.setState({mode: Modes.Reconfigure})}}>
                        Reconfigure Server
                    </button>
                    <button type="button" className="list-group-item list-group-item-action"
                            onClick={() => {this.setState({mode: Modes.Snapshot})}}>
                        Snapshot
                    </button>
                    <button type="button" className="list-group-item list-group-item-action"
                            onClick={() => {this.setState({mode: Modes.SystemInfo})}}>
                        System Information
                    </button>
                </div>
            )
        } else if (this.state.mode === Modes.Reconfigure) {
            controlItems.push(
                <ReconfigureMenu backToMenu={this.handleBackToMenu}
                                 showError={this.props.showError}
                                 key="reconfigure" />
            );
        } else if (this.state.mode === Modes.Snapshot) {
            controlItems.push(
                <Snapshot backToMenu={this.handleBackToMenu}
                          showError={this.props.showError}
                          key="snapshot" />
            );
        } else if (this.state.mode === Modes.SystemInfo) {
            controlItems.push(
                <SystemInfo backToMenu={this.handleBackToMenu}
                            key="systemInfo" />
            );
        }

        return (
            <div className="my-5 card mx-auto">
                <div className="card-body">
                    <form>
                        {controlItems}
                        <div className="d-md-block mt-3 text-end">
                            <button className="btn btn-danger bg-gradient"
                                    type="button"
                                    onClick={(e: React.MouseEvent) => {e.preventDefault(); fetch('/control/api/stop', {method: 'post'});}}>
                                <FaStop /> Stop
                            </button>
                        </div>
                    </form>
                </div>
            </div>
        );
    };
};
