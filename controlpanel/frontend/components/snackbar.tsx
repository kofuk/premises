import CloseIcon from '@mui/icons-material/Close';
import {IconButton, Snackbar} from '@mui/material';

export default ({message, onClose}: {message: string; onClose: () => void}) => {
  return (
    <Snackbar
      anchorOrigin={{vertical: 'top', horizontal: 'center'}}
      open={message.length > 0}
      autoHideDuration={10000}
      onClose={onClose}
      message={message}
      action={
        <>
          <IconButton aria-label="close" color="inherit" sx={{p: 0.5}} onClick={onClose}>
            <CloseIcon />
          </IconButton>
        </>
      }
    />
  );
};
