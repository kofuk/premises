import React from 'react';

import CloseIcon from '@mui/icons-material/Close';
import {IconButton, Snackbar as OriginalSnackbar} from '@mui/material';

const Snackbar = ({message, onClose}: {message: string; onClose: () => void}) => {
  return (
    <OriginalSnackbar
      anchorOrigin={{vertical: 'top', horizontal: 'center'}}
      open={message.length > 0}
      autoHideDuration={10000}
      onClose={onClose}
      message={message}
      action={
        <IconButton aria-label="close" color="inherit" sx={{p: 0.5}} onClick={onClose}>
          <CloseIcon />
        </IconButton>
      }
    />
  );
};

export default Snackbar;
