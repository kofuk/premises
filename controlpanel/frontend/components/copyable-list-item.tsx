import React from 'react';

import {MdContentCopy} from '@react-icons/all-files/md/MdContentCopy';
import {useTranslation} from 'react-i18next';

type Prop = {
  title: string;
  content: string;
};

const CopyableListItem = ({title, content}: Prop) => {
  const [t] = useTranslation();

  const handleCopy = () => {
    navigator.clipboard.writeText(content);
  };

  return (
    <div className="list-group-item">
      <h5 className="mb-1">{title}</h5>
      <div className="mb-1 d-flex flex-row">
        <div className="flex-fill">{content}</div>
        <div>
          <button type="button" className="btn btn-outline-dark btn-sm" onClick={handleCopy}>
            <MdContentCopy className="me-1" />
            {t('copy')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default CopyableListItem;
