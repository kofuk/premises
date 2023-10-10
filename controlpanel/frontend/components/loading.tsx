import {Box, Skeleton} from '@mui/material';

export default () => {
  return (
    <Box sx={{width: '70%', m: '0 auto'}}>
      <Skeleton animation="wave" height={200} />
      <Skeleton animation="wave" height={40} />
      <Skeleton animation="wave" height={40} width="80%" />
      <Skeleton animation="wave" height={40} width="95%" />
    </Box>
  );
};
