import * as React from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';

import '../../i18n';
import {t} from 'i18next';

type Prop = {
    backToMenu: () => void;
    showError: (message: string) => void;
};

type State = {
    isRequesting: boolean;
};

export default class Snapshot extends React.Component<Prop, State> {
    state: State = {
        isRequesting: false
    };

    handleSnapshot = () => {
        this.setState({isRequesting: true});
        fetch('/control/api/snapshot', {method: 'POST'})
            .then((resp) => resp.json())
            .then((resp) => {
                this.setState({isRequesting: false});
                if (!resp['success']) {
                    this.props.showError(resp['message']);
                    return;
                }
            });
    };

    render = () => {
        return (
            <div className="m-2">
                <button className="btn btn-outline-primary" onClick={this.props.backToMenu}>
                    <IoIosArrowBack /> {t('back')}
                </button>
                <div className="m-2">{t('snapshot_description')}</div>
                <div className="text-center">
                    <button type="button" className="btn btn-lg btn-primary" onClick={this.handleSnapshot} disabled={this.state.isRequesting}>
                        {this.state.isRequesting ? <div className="spinner-border spinner-border-sm me-2" role="status"></div> : <></>}
                        {this.state.isRequesting ? t('requesting') : t('take_snapshot')}
                    </button>
                </div>
            </div>
        );
    };
}
