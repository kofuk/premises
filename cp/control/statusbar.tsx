import * as React from 'react';

type State = {
    isError: boolean,
    message: string
};

export default class StatusBar extends React.Component<{isError: boolean, message: string}, {}> {
    render() {
        const appearance = this.props.isError ? 'alert-danger' : 'alert-success';
        return (
            <div className={`alert ${appearance}`}>
                {this.props.message}
            </div>
        );
    }
};
