import {CircularProgress} from '@mui/material';
import {Box} from '@mui/system';

const LoadingPage = () => {
  return (
    <Box sx={{mt: 12, textAlign: 'center'}}>
      <CircularProgress />
    </Box>
  );
};

export default LoadingPage;
