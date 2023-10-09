import {useState} from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';

import '@/i18n';
import {t} from 'i18next';

type Prop = {
  backToMenu: () => void;
  showError: (message: string) => void;
};

export default (props: Prop) => {
  const {backToMenu, showError} = props;
  const [isRequesting, setIsRequesting] = useState(false);

  const handleSnapshot = () => {
    setIsRequesting(true);
    fetch('/api/snapshot', {method: 'POST'})
      .then((resp) => resp.json())
      .then((resp) => {
        setIsRequesting(false);
        if (!resp['success']) {
          showError(resp['message']);
          return;
        }
      });
  };

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <IoIosArrowBack /> {t('back')}
      </button>
      <div className="m-2">{t('snapshot_description')}</div>
      <div className="text-center">
        <button type="button" className="btn btn-lg btn-primary" onClick={handleSnapshot} disabled={isRequesting}>
          {isRequesting ? <div className="spinner-border spinner-border-sm me-2" role="status"></div> : <></>}
          {isRequesting ? t('requesting') : t('take_snapshot')}
        </button>
      </div>
    </div>
  );
};