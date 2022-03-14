import * as React from 'react';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';

type Prop = ItemProp & {
    serverVersion: string,
    setServerVersion: (val: string) => void
};

type McVersion = {
    name: string,
    isStable: boolean,
    channel: string,
    releaseDate: string
};

type State = {
    mcVersions: McVersion[],
    showStable: boolean,
    showSnapshot: boolean,
    showBeta: boolean,
    showAlpha: boolean,
    refreshing: boolean
};

export default class ServerVersionConfigItem extends ConfigItem<Prop, State> {
    state: State = {
        mcVersions: [],
        showStable: true,
        showSnapshot: false,
        showAlpha: false,
        showBeta: false,
        refreshing: false
    };

    constructor(prop: Prop) {
        super(prop, 'Game Version');
    }

    componentDidMount() {
        fetch('/control/api/mcversions')
            .then(resp => resp.json())
            .then(resp => {
                this.setState({mcVersions: resp});
                this.postUpdateCondition();
            });
    }

    handleRefresh() {
        this.setState({refreshing: true});
        fetch('/control/api/mcversions?reload')
            .then(resp => resp.json())
            .then(resp => {
                this.setState({mcVersions: resp});
                this.postUpdateCondition();
                this.setState({refreshing: false});
            });
    }

    handleChange(val: string) {
        this.props.setServerVersion(val);
    }

    postUpdateCondition() {
        const versions = this.state.mcVersions
            .filter(e => this.state.showStable || (e.channel !== 'stable'))
            .filter(e => this.state.showSnapshot || (e.channel !== 'snapshot'))
            .filter(e => this.state.showBeta || (e.channel !== 'beta'))
            .filter(e => this.state.showAlpha || (e.channel !== 'alpha'));
        if (!versions.find(e => e.name === this.props.serverVersion)) {
            if (versions.length > 0) {
                this.props.setServerVersion(versions[0].name);
            } else {
                this.props.setServerVersion(this.state.mcVersions[0].name);
            }
        }
    }

    createContent(): React.ReactElement {
        const versions = this.state.mcVersions
            .filter(e => this.state.showStable || (e.channel !== 'stable'))
            .filter(e => this.state.showSnapshot || (e.channel !== 'snapshot'))
            .filter(e => this.state.showBeta || (e.channel !== 'beta'))
            .filter(e => this.state.showAlpha || (e.channel !== 'alpha'))
            .map(e => <option value={e.name} key={e.name}>{e.name}</option>);
        return (
            <>
                <select className="form-select" area-label="Choose A Version" value={this.props.serverVersion}
                        onChange={(e) => this.handleChange(e.target.value)}>
                    {versions}
                </select>
                <div className="m-1 text-end">
                    <button type="button" className="btn btn-sm btn-outline-secondary"
                            onClick={this.handleRefresh.bind(this)} disabled={this.state.refreshing}>
                        {this.state.refreshing
                             ? <div className="spinner-border spinner-border-sm me-1" role="status"></div>
                             : <IoIosRefresh />}
                        Refresh
                    </button>
                </div>
                <div className="m-1 form-check form-switch">
                    <input className="form-check-input" type="checkbox" id="showStable" checked={this.state.showStable}
                           onChange={() => {this.setState({showStable: !this.state.showStable}); this.postUpdateCondition();}} />
                    <label className="form-check-label" htmlFor="showStable">Show Stable</label>
                </div>
                <div className="m-1 form-check form-switch">
                    <input className="form-check-input" type="checkbox" id="showSnapshot" checked={this.state.showSnapshot}
                           onChange={() => {this.setState({showSnapshot: !this.state.showSnapshot}); this.postUpdateCondition();}} />
                    <label className="form-check-label" htmlFor="showSnapshot">Show Snapshot</label>
                </div>
                <div className="m-1 form-check form-switch">
                    <input className="form-check-input" type="checkbox" id="showSnapshot" checked={this.state.showBeta}
                           onChange={() => {this.setState({showBeta: !this.state.showBeta}); this.postUpdateCondition();}} />
                    <label className="form-check-label" htmlFor="showBeta">Show Beta</label>
                </div>
                <div className="m-1 form-check form-switch">
                    <input className="form-check-input" type="checkbox" id="showSnapshot" checked={this.state.showAlpha}
                           onChange={() => {this.setState({showAlpha: !this.state.showAlpha}); this.postUpdateCondition();}} />
                    <label className="form-check-label" htmlFor="showAlpha">Show Alpha</label>
                </div>
                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep}>Next</button>
                </div>
            </>
        );
    }
};
