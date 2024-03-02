import React, {ReactNode} from 'react';

import {Box, Typography} from '@mui/material';

import {ItemProp} from './prop';

const ConfigContainer = ({
  title,
  isFocused,
  requestFocus,
  stepNum,
  children
}: ItemProp & {
  title: string;
  children: ReactNode;
}) => {
  const titleStyle = {
    userSelect: 'none',
    cursor: 'pointer',
    borderRadius: 2,
    transition: 'background-color 300ms',
    '&:hover': {
      backgroundColor: 'rgba(0, 0, 0, 0.1)'
    },
    '&:active': {
      backgroundColor: 'rgba(0, 0, 0, 0.2)'
    }
  };

  return (
    <Box sx={{display: 'flex', m: 1}}>
      <Box sx={{my: 2}}>
        <svg height="30" version="1.1" viewBox="0 0 100 100" width="30" xmlns="http://www.w3.org/2000/svg">
          <circle cx="50" cy="50" fill={isFocused ? '#186fc7' : '#adadad'} r="50" />
          <text dominantBaseline="central" fill="white" fontFamily="sans-serif" fontSize="50" textAnchor="middle" x="50" y="45">
            {stepNum}
          </text>
        </svg>
      </Box>
      <Box sx={{mx: 1, p: 1, flex: 1, border: 'solid 1px #dbdbdb', borderRadius: 1}}>
        <Typography onClick={requestFocus} sx={titleStyle} variant="h4">
          {title}
        </Typography>
        {isFocused && <Box sx={{p: 1}}>{children}</Box>}
      </Box>
    </Box>
  );
};

export default ConfigContainer;
