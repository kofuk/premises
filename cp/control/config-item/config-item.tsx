import * as React from 'react';

import {ItemProp} from './prop';

export abstract class ConfigItem<Prop extends ItemProp, State> extends React.Component<Prop, State> {
    title: string;

    constructor(prop: Prop, title: string) {
        super(prop);
        this.title = title;
    }

    abstract createContent(): React.ReactElement;

    render = () => {
        let mainContent: React.ReactElement;
        if (this.props.isFocused) {
            mainContent = this.createContent();
        } else {
            mainContent = <></>;
        }

        return (
            <div className="d-flex flex-row m-2">
                <div className="my-2">
                    <svg width="30" height="30" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" version="1.1">
                        <circle cx="50" cy="50" r="50" fill={this.props.isFocused ? 'blue' : 'gray'} />
                        <text x="50" y="45" textAnchor="middle" dominantBaseline="central" fontFamily="sans-serif" fontSize="50" fill="white">
                            {this.props.stepNum}
                        </text>
                    </svg>
                </div>
                <div className="mx-2 p-2 flex-fill border rounded">
                    <h3 className="step-title user-select-none" onClick={this.props.requestFocus}>{this.title}</h3>
                    {mainContent}
                </div>
            </div>
        );
    };
};
