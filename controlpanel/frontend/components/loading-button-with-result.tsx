import React from 'react';

import {Check as CheckIcon} from '@mui/icons-material';
import {LoadingButton} from '@mui/lab';
import {green} from '@mui/material/colors';

type Props = {
  type?: any;
  variant?: any;
  disabled?: boolean;
  loading?: boolean;
  success?: boolean;
  children: React.ReactNode;
};

const LoadingButtonWithResult = ({type, variant, disabled, loading, success, children}: Props) => {
  const sx = {
    ...(success && {
      bgcolor: green[500],
      '&:hover': {
        bgcolor: green[700]
      }
    })
  };

  return (
    <LoadingButton type={type} variant={variant} disabled={disabled} loading={loading} sx={sx}>
      {success ? <CheckIcon /> : children}
    </LoadingButton>
  );
};

export default LoadingButtonWithResult;
