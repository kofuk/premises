import * as React from 'react';

type Prop = {
    backToMenu: () => void
};

type SystemInfoData = {
    premisesVersion: string,
    hostOS: string
} | null;

type State = {
    systemInfo: SystemInfoData
};

export default class SystemInfo extends React.Component<Prop, State> {
    state: State = {
        systemInfo: null
    };

    componentDidMount() {
        fetch('/control/api/systeminfo')
            .then(resp => resp.json())
            .then(resp => {
                this.setState({systemInfo: resp});
            });
    }

    render() {
        let mainContents: React.ReactElement;
        if (this.state.systemInfo === null) {
            mainContents = <></>;
        } else {
            mainContents = (
                <div className="list-group">
                    <div className="list-group-item">
                        <h5 className="mb-1">Server Version</h5>
                        <p className="mb-1">{this.state.systemInfo.premisesVersion}</p>
                    </div>
                    <div className="list-group-item">
                        <h5 className="mb-1">Host OS</h5>
                        <p className="mb-1">{this.state.systemInfo.hostOS}</p>
                    </div>
                </div>
            );
        }

        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    Back
                </button>
                <div className="m-2">
                    {mainContents}
                </div>
            </div>
        );
    }
};
