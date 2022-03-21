import * as React from 'react';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';
import {WorldBackup} from './world-backup';

type Prop = ItemProp & {
    worldName: string;
    backupGeneration: string;
    useCachedWorld: boolean;
    setWorldName: (val: string) => void;
    setBackupGeneration: (val: string) => void;
    setUseCachedWorld: (val: boolean) => void;
};

type State = {
    backups: WorldBackup[];
    refreshing: boolean;
};

export default class ChooseBackupConfigItem extends ConfigItem<Prop, {}> {
    state: State = {
        backups: [],
        refreshing: false
    };

    constructor(prop: Prop) {
        super(prop, t('config_choose_backup'));
    }

    componentDidMount = () => {
        fetch('/control/api/backups')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({backups: resp});
                if (resp.length > 0) {
                    this.props.setWorldName(resp[0].worldName);
                    this.props.setBackupGeneration(resp[0].generations[0].id);
                }
            });
    };

    handleRefresh = () => {
        this.setState({refreshing: true});
        fetch('/control/api/backups?reload')
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({backups: resp});
                if (resp.length > 0) {
                    this.props.setWorldName(resp[0].worldName);
                    this.props.setBackupGeneration(resp[0].generations[0].id);
                }
                this.setState({refreshing: false});
            });
    };

    handleChangeWorld = (worldName: string) => {
        this.props.setWorldName(worldName);
        const generations = this.state.backups.find((e) => e.worldName === worldName)!.generations;
        this.props.setBackupGeneration(generations[0].id);
    };

    handleChangeGeneration = (generationId: string) => {
        this.props.setBackupGeneration(generationId);
    };

    createBackupSelector = (): React.ReactElement => {
        const worlds = (
            <div className="m-2">
                <label className="form-label" htmlFor="worldSelect">
                    {t('select_world')}
                </label>
                <select
                    className="form-select"
                    value={this.props.worldName}
                    id="worldSelect"
                    onChange={(e) => this.handleChangeWorld(e.target.value)}
                >
                    {this.state.backups.map((e) => (
                        <option value={e.worldName} key={e.worldName}>
                            {e.worldName}
                        </option>
                    ))}
                </select>
            </div>
        );
        const worldData = this.state.backups.find((e) => e.worldName === this.props.worldName);
        const generations = worldData ? (
            <div className="m-2">
                <label className="form-label" htmlFor="backupGenerationSelect">
                    {t('backup_generation')}
                </label>
                <select
                    className="form-select"
                    value={this.props.backupGeneration}
                    id="backupGenerationSelect"
                    onChange={(e) => this.handleChangeGeneration(e.target.value)}
                >
                    {worldData.generations.map((e) => {
                        const dateTime = new Date(e.timestamp);
                        return (
                            <option value={e.id} key={e.gen}>
                                {(e.gen == 'latest' ? 'Latest' : `${e.gen} gen ago`) + ` (${dateTime.toLocaleString()})`}
                            </option>
                        );
                    })}
                </select>
            </div>
        ) : (
            <></>
        );

        return (
            <>
                {worlds}
                {generations}
                <div className="m-2 form-check form-switch">
                    <input
                        className="form-check-input"
                        type="checkbox"
                        id="useCachedWorld"
                        checked={this.props.useCachedWorld}
                        onChange={(e) => this.props.setUseCachedWorld(e.target.checked)}
                    />
                    <label className="form-check-label" htmlFor="useCachedWorld">
                        {t('use_cached_world')}
                    </label>
                </div>
                <div className="m-1">
                    <button type="button" className="btn btn-sm btn-outline-secondary" onClick={this.handleRefresh} disabled={this.state.refreshing}>
                        {this.state.refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
                        {t('refresh')}
                    </button>
                </div>
            </>
        );
    };

    createEmptyMessage = (): React.ReactElement => {
        return (
            <div className="alert alert-warning" role="alert">
                {t('no_backups')}
            </div>
        );
    };

    createContent = (): React.ReactElement => {
        const content = this.state.backups.length === 0 ? this.createEmptyMessage() : this.createBackupSelector();
        return (
            <>
                {content}
                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep} disabled={this.state.backups.length === 0}>
                        {t('next')}
                    </button>
                </div>
            </>
        );
    };
}
