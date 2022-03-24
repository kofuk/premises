import * as React from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

import '../../i18n';
import {t} from 'i18next';

import CopyableListItem from '../../components/copyable-list-item';

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
        refreshing: true
    };

    componentDidMount = () => {
        fetch('/control/api/worldinfo')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({worldInfo: resp, refreshing: false});
            });
    };

    handleRefresh = () => {
        this.setState({refreshing: true});
        fetch('/control/api/worldinfo')
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
                    <CopyableListItem title={t('world_info_game_version')} content={this.state.worldInfo.serverVersion} />
                    <CopyableListItem title={t('world_info_world_name')} content={this.state.worldInfo.world.name} />
                    <CopyableListItem title={t('world_info_seed')} content={this.state.worldInfo.world.seed} />
                </div>
            );
        }

        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    <IoIosArrowBack /> {t('back')}
                </button>
                <div className="m-2">{mainContents}</div>
                <div className="m-1">
                    <button type="button" className="btn btn-sm btn-outline-secondary" onClick={this.handleRefresh} disabled={this.state.refreshing}>
                        {this.state.refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
                        {t('refresh')}
                    </button>
                </div>
            </div>
        );
    };
}
