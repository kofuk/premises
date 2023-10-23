import React from 'react';

import {Box, Skeleton} from '@mui/material';

type Props = {
  compact?: boolean | undefined;
};

const Loading = ({compact}: Props) => {
  if (compact) {
    return (
      <Box sx={{m: '10px auto'}}>
        <Skeleton animation="wave" height={50} width="40%" />
        <Skeleton animation="wave" height={20} width="70%" />
        <Skeleton animation="wave" height={20} width="55%" />
        <Skeleton animation="wave" height={20} width="60%" />
      </Box>
    );
  }
  return (
    <Box sx={{width: '70%', m: '0 auto'}}>
      <Skeleton animation="wave" height={200} />
      <Skeleton animation="wave" height={40} />
      <Skeleton animation="wave" height={40} width="80%" />
      <Skeleton animation="wave" height={40} width="95%" />
    </Box>
  );
};

export default Loading;
