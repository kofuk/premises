import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';
import {LoadingButton} from '@mui/lab';
import {Stack} from '@mui/material';
import {Box} from '@mui/system';

type Prop = {
  backToMenu: () => void;
};

const QuickUndo = ({backToMenu}: Prop) => {
  const [t] = useTranslation();

  const handleSnapshot = () => {
    (async () => {
      try {
        const result = await fetch('/api/quickundo/snapshot', {method: 'POST'}).then((resp) => resp.json());
        if (!result['success']) {
          throw new Error(t(`error.code_${result['errorCode']}`));
        }
      } catch (err) {
        console.error(err);
      }
    })();
  };

  const [revertConfirming, setRevertConfirming] = useState(false);

  const handleUndo = () => {
    if (!revertConfirming) {
      setRevertConfirming(true);
      return;
    }
    setRevertConfirming(false);

    (async () => {
      try {
        const result = await fetch('/api/quickundo/undo', {method: 'POST'}).then((resp) => resp.json());
        if (!result['success']) {
          throw new Error(t(`error.code_${result['errorCode']}`));
        }
      } catch (err) {
        console.error(err);
      }
    })();
  };

  return (
    <Box sx={{m: 2}}>
      <button className="btn btn-outline-primary" onClick={backToMenu}>
        <ArrowBackIcon /> {t('back')}
      </button>
      <Box sx={{m: 2}}>{t('snapshot_description')}</Box>
      <Stack direction="row" justifyContent="center" spacing={1}>
        <LoadingButton onClick={handleSnapshot} type="button" variant="contained">
          {t('take_snapshot')}
        </LoadingButton>
        <LoadingButton onClick={handleUndo} type="button" variant="outlined">
          {revertConfirming ? t('revert_snapshot_confirm') : t('revert_snapshot')}
        </LoadingButton>
      </Stack>
    </Box>
  );
};

export default QuickUndo;
