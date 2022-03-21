import * as React from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';

import CopyableListItem from '../../components/copyable-list-item';

type Prop = {
    backToMenu: () => void;
};

type SystemInfoData = {
    premisesVersion: string;
    hostOS: string;
} | null;

type State = {
    systemInfo: SystemInfoData;
};

export default class SystemInfo extends React.Component<Prop, State> {
    state: State = {
        systemInfo: null
    };

    componentDidMount = () => {
        fetch('/control/api/systeminfo')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({systemInfo: resp});
            });
    };

    render = () => {
        let mainContents: React.ReactElement;
        if (this.state.systemInfo === null) {
            mainContents = <></>;
        } else {
            mainContents = (
                <div className="list-group">
                    <CopyableListItem title="Server Version" content={this.state.systemInfo.premisesVersion} />
                    <CopyableListItem title="Host OS" content={this.state.systemInfo.hostOS} />
                </div>
            );
        }

        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    <IoIosArrowBack /> Back
                </button>
                <div className="m-2">{mainContents}</div>
            </div>
        );
    };
}
