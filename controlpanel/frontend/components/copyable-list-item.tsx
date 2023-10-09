import {MdContentCopy} from '@react-icons/all-files/md/MdContentCopy';

import '../i18n';
import {t} from 'i18next';

type Prop = {
  title: string;
  content: string;
};

export default (props: Prop) => {
  const {title, content} = props;

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
