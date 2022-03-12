import * as React from 'react';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';
import {WorldBackup} from './world-backup';

type Prop = ItemProp & {
    worldName: string,
    backupGeneration: string,
    useCachedWorld: boolean,
    setWorldName: (val: string) => void,
    setBackupGeneration: (val: string) => void,
    setUseCachedWorld: (val: boolean) => void
};

type State = {
    backups: WorldBackup[]
};

export default class ChooseBackupConfigItem extends ConfigItem<Prop, {}> {
    state: State = {
        backups: []
    };

    constructor(prop: Prop) {
        super(prop, 'World');
    }

    componentDidMount() {
        fetch('/control/api/backups')
            .then(resp => resp.json())
            .then(resp => {
                this.setState({backups: resp});
                if (resp.length > 0) {
                    this.props.setWorldName(resp[0].worldName);
                    this.props.setBackupGeneration(resp[0].generations[0])
                }
            });
    }

    handleChangeWorld(worldName: string) {
        this.props.setWorldName(worldName);
        const generations = this.state.backups.find(e => e.worldName === worldName)!.generations;
        this.props.setBackupGeneration(generations[0]);
    }

    handleChangeGeneration(generation: string) {
        this.props.setBackupGeneration(generation);
    }

    createBackupSelector(): React.ReactElement {
        const worlds = (
            <div className="m-2">
                <label className="form-label" htmlFor="worldSelect">World</label>
                <select className="form-select" value={this.props.worldName} id="worldSelect"
                        onChange={(e) => this.handleChangeWorld(e.target.value)}>
                    {this.state.backups.map(e => <option value={e.worldName} key={e.worldName}>{e.worldName}</option>)}
                </select>
            </div>
        );
        const worldData = this.state.backups.find(e => e.worldName === this.props.worldName);
        const generations = worldData ? (
            <div className="m-2">
                <label className="form-label" htmlFor="backupGenerationSelect">Backup Generation</label>
                <select className="form-select" value={this.props.backupGeneration} id="backupGenerationSelect"
                        onChange={(e) => this.handleChangeGeneration(e.target.value)}>
                    {worldData.generations.map(e => <option value={e} key={e}>{e == 'latest' ? 'Latest' : `${e} gen ago`}</option>)}
                </select>
            </div>
        ) : <></>;

        return (
            <>
                {worlds}
                {generations}
                <div className="m-2 form-check form-switch">
                    <input className="form-check-input" type="checkbox" id="useCachedWorld" checked={this.props.useCachedWorld}
                           onChange={(e) => this.props.setUseCachedWorld(e.target.checked)}/>
                    <label className="form-check-label" htmlFor="useCachedWorld">Use Cached World Data If Possible</label>
                </div>
            </>
        );
    }

    createEmptyMessage(): React.ReactElement {
        return (
            <div className="alert alert-warning" role="alert">
                No world backups found. Please generate a new world.
            </div>
        );
    }

    createContent(): React.ReactElement {
        const content = this.state.backups.length === 0 ? this.createEmptyMessage() : this.createBackupSelector();
        return (
            <>
                {content}
                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep}
                            disabled={this.state.backups.length === 0}>
                        Next
                    </button>
                </div>
            </>
        );
    }
};
