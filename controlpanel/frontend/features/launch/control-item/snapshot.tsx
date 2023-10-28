import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';

type Prop = {
  backToMenu: () => void;
  showError: (message: string) => void;
};

const Snapshot = (props: Prop) => {
  const [t] = useTranslation();

  const {backToMenu, showError} = props;
  const [isRequesting, setIsRequesting] = useState(false);

  const handleSnapshot = () => {
    (async () => {
      setIsRequesting(true);
      try {
        const result = await fetch('/api/snapshot', {method: 'POST'}).then((resp) => resp.json());
        if (!result['success']) {
          showError(result['reason']);
          return;
        }
      } catch (err) {
        console.error(err);
      } finally {
        setIsRequesting(false);
      }
    })();
  };

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <ArrowBackIcon /> {t('back')}
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

export default Snapshot;
