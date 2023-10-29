import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';

type Prop = {
  backToMenu: () => void;
  showError: (message: string) => void;
};

const QuickUndo = (props: Prop) => {
  const [t] = useTranslation();

  const {backToMenu, showError} = props;
  const [isRequesting, setIsRequesting] = useState(false);

  const handleSnapshot = () => {
    (async () => {
      setIsRequesting(true);
      try {
        const result = await fetch('/api/quickundo/snapshot', {method: 'POST'}).then((resp) => resp.json());
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

  const handleUndo = () => {
    (async () => {
      setIsRequesting(true);
      try {
        const result = await fetch('/api/quickundo/undo', {method: 'POST'}).then((resp) => resp.json());
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
      <div className="m-2">簡易的なスナップショットで素早くある時点のワールドに戻ります</div>
      <div className="alert alert-warning">
        2 回目以降のスナップショットの作成は、前回のスナップショットを削除して行われます。つまり、この機能では 1
        段階しかワールドを巻き戻すことができません。
        <br />
      </div>
      <div className="alert alert-warning">
        このスナップショットはサーバの再設定後も保持されるため、ワールドのデータを別のワールドのデータで上書きしないよう注意してください。
      </div>
      <div className="text-center">
        <button className="btn btn-lg btn-primary" disabled={isRequesting} onClick={handleSnapshot} type="button">
          {isRequesting ? <div className="spinner-border spinner-border-sm me-2" role="status"></div> : <></>}
          {isRequesting ? t('requesting') : t('take_snapshot')}
        </button>
        <button className="btn btn-lg btn-primary" disabled={isRequesting} onClick={handleUndo} type="button">
          スナップショットに戻す
        </button>
      </div>
    </div>
  );
};

export default QuickUndo;
