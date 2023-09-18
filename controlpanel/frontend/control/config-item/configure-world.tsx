import * as React from 'react';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';

type Prop = ItemProp & {
    levelType: LevelType;
    seed: string;
    setLevelType: (val: LevelType) => void;
    setSeed: (val: string) => void;
};

export enum LevelType {
    Default = 'default',
    Superflat = 'flat',
    LargeBiomes = 'largeBiomes',
    Amplified = 'amplified'
}

class LevelTypeInfo {
    levelType: LevelType;
    label: string;

    constructor(levelType: LevelType, label: string) {
        this.levelType = levelType;
        this.label = label;
    }

    createReactElement = (): React.ReactElement => {
        return <option value={this.levelType}>{this.label}</option>;
    };
}

const levelTypes: LevelTypeInfo[] = [
    new LevelTypeInfo(LevelType.Default, t('world_type_default')),
    new LevelTypeInfo(LevelType.Superflat, t('world_type_superflat')),
    new LevelTypeInfo(LevelType.LargeBiomes, t('world_type_large_biomes')),
    new LevelTypeInfo(LevelType.Amplified, t('world_type_amplified'))
];

export default class ConfigureWorldConfigItem extends ConfigItem<Prop, {}> {
    constructor(prop: Prop) {
        super(prop, t('config_configure_world'));
    }

    createContent = (): React.ReactElement => {
        return (
            <>
                <div className="m-2">
                    <label className="form-label" htmlFor="seed">
                        {t('seed')}
                    </label>
                    <input className="form-control" value={this.props.seed} id="seed" onChange={(e) => this.props.setSeed(e.target.value)} />
                </div>

                <div className="m-2">
                    <label className="form-label" htmlFor="selectLevelType">
                        {t('world_type')}
                    </label>
                    <select
                        className="form-select"
                        value={this.props.levelType}
                        id="selectLeveltype"
                        onChange={(e) => this.props.setLevelType(e.target.value as LevelType)}
                    >
                        {levelTypes.map((e) => e.createReactElement())}
                    </select>
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
