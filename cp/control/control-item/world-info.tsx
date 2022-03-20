import * as React from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

type Prop = {
    backToMenu: () => void;
};

type WorldDetail = {
    name: string;
    seed: string;
};

type WorldInfoData = {
    serverVersion: string;
    world: WorldDetail;
} | null;

type State = {
    worldInfo: WorldInfoData;
    refreshing: boolean;
};

export default class WorldInfo extends React.Component<Prop, State> {
    state: State = {
        worldInfo: null,
        refreshing: false
    };

    componentDidMount = () => {
        fetch('/control/api/worldinfo')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({worldInfo: resp});
            });
    };

    handleRefresh = () => {
        this.setState({refreshing: true});
        fetch('/control/api/worldinfo?reload')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({worldInfo: resp, refreshing: false});
            });
    };

    render = () => {
        let mainContents: React.ReactElement;
        if (this.state.worldInfo === null) {
            mainContents = <></>;
        } else {
            mainContents = (
                <div className="list-group">
                    <div className="list-group-item">
                        <h5 className="mb-1">Game Version</h5>
                        <p className="mb-1">{this.state.worldInfo.serverVersion}</p>
                    </div>
                    <div className="list-group-item">
                        <h5 className="mb-1">World</h5>
                        <p className="mb-1">{this.state.worldInfo.world.name}</p>
                    </div>
                    <div className="list-group-item">
                        <h5 className="mb-1">Seed</h5>
                        <p className="mb-1">{this.state.worldInfo.world.seed}</p>
                    </div>
                </div>
            );
        }

        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    <IoIosArrowBack /> Back
                </button>
                <div className="m-2">{mainContents}</div>
                <div className="m-1">
                    <button type="button" className="btn btn-sm btn-outline-secondary" onClick={this.handleRefresh} disabled={this.state.refreshing}>
                        {this.state.refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
                        Refresh
                    </button>
                </div>
            </div>
        );
    };
}
