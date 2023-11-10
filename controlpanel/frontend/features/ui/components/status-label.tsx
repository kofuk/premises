import React from 'react';

import {Typography} from '@mui/material';

type Prop = {
  message: string;
  progress: number;
};

const StatusLabel = ({message, progress}: Prop) => {
  const sx = {
    background: `linear-gradient(90deg, #6aa5eb 0%, #6aa5eb ${progress}%, #93c0f5 ${progress}%, #93c0f5 100%)`,
    color: 'black',
    width: 500,
    padding: '5px 30px',
    borderRadius: 1000,
    border: 'solid 1px #99c1f0'
  };

  return (
    <Typography component="div" sx={sx}>
      {message}
    </Typography>
  );
};

export default StatusLabel;
