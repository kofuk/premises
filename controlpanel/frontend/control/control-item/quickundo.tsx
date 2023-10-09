import {useState} from 'react';
import {IoIosArrowBack} from '@react-icons/all-files/io/IoIosArrowBack';

import '../../i18n';
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
    fetch('/api/quickundo/snapshot', {method: 'POST'})
      .then((resp) => resp.json())
      .then((resp) => {
        setIsRequesting(false);
        if (!resp['success']) {
          showError(resp['message']);
          return;
        }
      });
  };

  const handleUndo = () => {
    setIsRequesting(true);
    fetch('/api/quickundo/undo', {method: 'POST'})
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
        <button type="button" className="btn btn-lg btn-primary" onClick={handleSnapshot} disabled={isRequesting}>
          {isRequesting ? <div className="spinner-border spinner-border-sm me-2" role="status"></div> : <></>}
          {isRequesting ? t('requesting') : t('take_snapshot')}
        </button>
        <button type="button" className="btn btn-lg btn-primary" onClick={handleUndo} disabled={isRequesting}>
          スナップショットに戻す
        </button>
      </div>
    </div>
  );
};
