import {Typography} from '@mui/material';
import {useTranslation} from 'react-i18next';

const WarningText = ({children}: {children: string}) => (
  <Typography component="span" sx={{color: '#b33930', fontWeight: 'bold'}} variant="body2">
    {children}
  </Typography>
);

export const valueLabel = (value: string | undefined | null, formatter?: (value: string) => string | undefined): React.ReactNode => {
  const [t] = useTranslation();
  if (value == null) {
    return <WarningText>{t('launch.not_set')}</WarningText>;
  }
  if (formatter) {
    return formatter(value) || value;
  }
  return value;
};
