import React from 'react';

import {useTranslation} from 'react-i18next';

import {Typography} from '@mui/material';

const WarningText = ({children}: {children: string}) => (
  <Typography component="span" sx={{color: '#b33930', fontWeight: 'bold'}} variant="body2">
    {children}
  </Typography>
);

export const valueLabel = (value: string | undefined | null, formatter?: (value: string) => string | undefined): React.ReactNode => {
  const [t] = useTranslation();
  if (value == null) {
    return <WarningText>{t('value_not_set')}</WarningText>;
  }
  if (formatter) {
    return formatter(value) || value;
  }
  return value;
};
