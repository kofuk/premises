import * as React from 'react';

import {ItemProp} from './prop';
import {ConfigItem} from './config-item';

type Prop = ItemProp & {
    machineType: string;
    setMachineType: (val: string) => void;
};

class Machine {
    name: string;
    memSize: number;
    nCores: number;
    price: number;

    constructor(name: string, memSize: number, nCores: number, price: number) {
        this.name = name;
        this.memSize = memSize;
        this.nCores = nCores;
        this.price = price;
    }

    getLabel = (): string => {
        return `${this.memSize}GB RAM & ${this.nCores}-core CPU, ¥${this.price}/h`;
    };

    createReactElement = (selectedValue: string, clickHandler: (val: string) => void): React.ReactElement => {
        return (
            <React.Fragment key={this.name}>
                <input
                    type="radio"
                    className="btn-check"
                    id={`machineType_${this.name}`}
                    name="machine-type"
                    autoComplete="off"
                    value={this.name}
                    checked={this.name === selectedValue}
                    onChange={() => clickHandler(this.name)}
                />
                <label className="btn btn-outline-primary" htmlFor={`machineType_${this.name}`} title={this.getLabel()}>
                    {this.memSize}GB
                </label>
            </React.Fragment>
        );
    };
}

const machines: Machine[] = [
    new Machine('2g', 2, 3, 3.3),
    new Machine('4g', 4, 4, 6.6),
    new Machine('8g', 8, 6, 13.2),
    new Machine('16g', 16, 8, 24.2),
    new Machine('32g', 32, 12, 48),
    new Machine('64g', 64, 24, 96.8)
];

export default class MachineTypeConfigItem extends ConfigItem<Prop, {}> {
    constructor(prop: Prop) {
        super(prop, 'Machine Type');
    }

    handleClick = (val: string) => {
        this.props.setMachineType(val);
    };

    createContent = (): React.ReactElement => {
        return (
            <>
                <div className="btn-group ms-3" role="group">
                    {machines.map((e) => e.createReactElement(this.props.machineType, this.handleClick))}
                </div>
                <div className="m-1 text-end">
                    <button type="button" className="btn btn-primary" onClick={this.props.nextStep}>
                        Next
                    </button>
                </div>
            </>
        );
    };
}
