import React from 'react';

import CloseIcon from '@mui/icons-material/Close';
import {IconButton, Snackbar as MuiSnackbar} from '@mui/material';

const Snackbar = ({message, onClose}: {message: string; onClose: () => void}) => {
  return (
    <MuiSnackbar
      action={
        <IconButton aria-label="close" color="inherit" onClick={onClose} sx={{p: 0.5}}>
          <CloseIcon />
        </IconButton>
      }
      anchorOrigin={{vertical: 'top', horizontal: 'center'}}
      autoHideDuration={10000}
      message={message}
      onClose={onClose}
      open={message.length > 0}
    />
  );
};

export default Snackbar;
