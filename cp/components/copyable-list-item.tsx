import * as React from 'react';
import {MdContentCopy} from '@react-icons/all-files/md/MdContentCopy';

import '../i18n';
import {t} from 'i18next';

type Prop = {
    title: string;
    content: string;
};

export default class CopyableListItem extends React.Component<Prop, {}> {
    handleCopy = () => {
        navigator.clipboard.writeText(this.props.content);
    };

    render = () => {
        return (
            <div className="list-group-item">
                <h5 className="mb-1">{this.props.title}</h5>
                <div className="mb-1 d-flex flex-row">
                    <div className="flex-fill">{this.props.content}</div>
                    <div>
                        <button type="button" className="btn btn-outline-dark btn-sm" onClick={this.handleCopy}>
                            <MdContentCopy className="me-1" />
                            {t('copy')}
                        </button>
                    </div>
                </div>
            </div>
        );
    };
}
