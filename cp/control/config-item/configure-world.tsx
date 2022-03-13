import * as React from 'react';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';
import {WorldBackup} from './world-backup';

type Prop = ItemProp & {
    levelType: LevelType,
    seed: string,
    setLevelType: (val: LevelType) => void,
    setSeed: (val: string) => void
};

export enum LevelType {
    Default = 'default',
    Superflat = 'superflat',
    LargeBiomes = 'largeBiomes',
    Amplified  = 'amplified'
};

class LevelTypeInfo {
    levelType: LevelType;
    label: string;

    constructor(levelType: LevelType, label: string) {
        this.levelType = levelType;
        this.label = label;
    }

    createReactElement(): React.ReactElement {
        return <option value={this.levelType}>{this.label}</option>;
    }
};

const levelTypes: LevelTypeInfo[] = [
    new LevelTypeInfo(LevelType.Default, 'Default'),
    new LevelTypeInfo(LevelType.Superflat, 'Superflat'),
    new LevelTypeInfo(LevelType.LargeBiomes, 'Large Biomes'),
    new LevelTypeInfo(LevelType.Amplified, 'Amplified')
];

export default class ConfigureWorldConfigItem extends ConfigItem<Prop, {}> {
    constructor(prop: Prop) {
        super(prop, 'Configure World');
    }

    createContent(): React.ReactElement {
        return (
            <>
                <div className="m-2">
                    <label className="form-label" htmlFor="seed">Seed</label>
                    <input className="form-control" value={this.props.seed} id="seed"
                           onChange={(e) => this.props.setSeed(e.target.value)} />
                </div>

                <div className="m-2">
                    <label className="form-label" htmlFor="selectLevelType">World Type</label>
                    <select className="form-select" value={this.props.levelType} id="selectLeveltype"
                            onChange={(e) => this.props.setLevelType(e.target.value as LevelType)}>
                        {levelTypes.map(e => e.createReactElement())}
                    </select>
                </div>

                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep}>Next</button>
                </div>
            </>
        );
    }
};
