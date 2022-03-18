import * as React from 'react';

type Prop = {
    isError: boolean;
    message: string;
};

export default class StatusBar extends React.Component<Prop, {}> {
    render = () => {
        const appearance = this.props.isError ? 'alert-danger' : 'alert-success';
        return <div className={`alert ${appearance}`}>{this.props.message}</div>;
    };
}
