import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';

export enum WorldLocation {
    Backups = 'backups',
    NewWorld = 'new-world'
}

type Prop = ItemProp & {
    worldSource: WorldLocation;
    setWorldSource: (val: WorldLocation) => void;
};

export default class WorldSourceConfigItem extends ConfigItem<Prop, {}> {
    constructor(prop: Prop) {
        super(prop, t('config_world_source'));
    }

    handleChange = (val: string) => {
        this.props.setWorldSource(val === 'backups' ? WorldLocation.Backups : WorldLocation.NewWorld);
    };

    createContent = (): React.ReactElement => {
        return (
            <>
                <div className="form-check">
                    <input
                        className="form-check-input"
                        type="radio"
                        name="worldSource"
                        value="backups"
                        id="worldSourceBackups"
                        checked={this.props.worldSource === WorldLocation.Backups}
                        onChange={(e) => this.handleChange(e.target.value)}
                    />
                    <label className="form-check-label" htmlFor="worldSourceBackups">
                        {t('use_backups')}
                    </label>
                </div>
                <div className="form-check">
                    <input
                        className="form-check-input"
                        type="radio"
                        name="worldSource"
                        value="newWorld"
                        id="worldSourceNewWorld"
                        checked={this.props.worldSource === WorldLocation.NewWorld}
                        onChange={(e) => this.handleChange(e.target.value)}
                    />
                    <label className="form-check-label" htmlFor="worldSourceNewWorld">
                        {t('generate_world')}
                    </label>
                </div>

                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep}>
                        {t('next')}
                    </button>
                </div>
            </>
        );
    };
}
