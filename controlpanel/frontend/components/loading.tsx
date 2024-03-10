import React from 'react';

import {Box} from '@mui/material';

import DelayedSkeleton from './delayed-skeleton';

type Props = {
  compact?: boolean | undefined;
};

const Loading = ({compact}: Props) => {
  if (compact) {
    return (
      <Box sx={{m: '10px auto'}}>
        <DelayedSkeleton height={50} width="40%" />
        <DelayedSkeleton height={20} width="70%" />
        <DelayedSkeleton height={20} width="55%" />
        <DelayedSkeleton height={20} width="60%" />
      </Box>
    );
  }
  return (
    <Box sx={{width: '70%', m: '0 auto'}}>
      <DelayedSkeleton height={200} />
      <DelayedSkeleton height={40} />
      <DelayedSkeleton height={40} width="80%" />
      <DelayedSkeleton height={40} width="95%" />
    </Box>
  );
};

export default Loading;
