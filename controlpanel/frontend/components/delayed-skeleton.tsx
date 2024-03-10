import React from 'react';

import {Fade, Skeleton} from '@mui/material';

type Props = {
  height?: number | string;
  width?: number | string;
  children?: React.ReactNode;
};

const DelayedSkeleton = (props: Props) => {
  return (
    <Fade in={true} timeout={1000}>
      <Skeleton animation="wave" {...props} />
    </Fade>
  );
};

export default DelayedSkeleton;
