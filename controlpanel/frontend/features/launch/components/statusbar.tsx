import React from 'react';

import {Alert} from '@mui/material';

type Prop = {
  message: string;
  progress: number;
};

const StatusBar = ({message, progress}: Prop) => {
  return (
    <Alert
      icon={false}
      severity="info"
      sx={{
        background: `linear-gradient(90deg, #115293 0%, #115293 ${progress}%, #1976d2 ${progress}%, #1976d2 100%)`
      }}
      variant="filled"
    >
      {message}
    </Alert>
  );
};

export default StatusBar;
