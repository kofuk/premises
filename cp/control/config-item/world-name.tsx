import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';
import {WorldBackup} from './world-backup';

type Prop = ItemProp & {
    worldName: string;
    setWorldName: (val: string) => void;
};

type State = {
    backups: WorldBackup[];
    duplicateName: boolean;
    invalidName: boolean;
};

export default class WorldNameConfigItem extends ConfigItem<Prop, State> {
    state: State = {
        backups: [],
        duplicateName: false,
        invalidName: false
    };

    constructor(prop: Prop) {
        super(prop, t('config_world_name'));
    }

    componentDidMount = () => {
        fetch('/control/api/backups')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({backups: resp});
            });
    };

    handleChange = (val: string) => {
        this.props.setWorldName(val);

        if (!val.match(/^[- _a-zA-Z0-9()]+$/)) {
            this.setState({invalidName: true});
            return;
        }
        if (this.state.backups.find((e) => e.worldName === val)) {
            this.setState({duplicateName: true});
            return;
        }

        this.setState({duplicateName: false, invalidName: false});
    };

    createContent = (): React.ReactElement => {
        let alert = <></>;
        if (this.state.invalidName) {
            alert = (
                <div className="m-2 alert alert-danger" role="alert">
                    Name must be alphanumeric.
                </div>
            );
        } else if (this.state.duplicateName) {
            alert = (
                <div className="m-2 alert alert-danger" role="alert">
                    World name duplicates.
                </div>
            );
        }

        return (
            <>
                <label className="form-label" htmlFor="newWorldName">
                    {t('world_name')}
                </label>
                <input
                    type="text"
                    className="form-control"
                    id="newWorldName"
                    value={this.props.worldName}
                    onChange={(e) => {
                        this.handleChange(e.target.value);
                    }}
                />
                {alert}
                <div className="m-1 text-end">
                    <button
                        type="button"
                        className="btn btn-primary"
                        onClick={this.props.nextStep}
                        disabled={this.props.worldName.length === 0 || this.state.duplicateName || this.state.invalidName}
                    >
                        {t('next')}
                    </button>
                </div>
            </>
        );
    };
}
