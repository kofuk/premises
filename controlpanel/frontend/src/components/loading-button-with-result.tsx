import {Check as CheckIcon} from '@mui/icons-material';
import {Button} from '@mui/material';
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
    <Button disabled={disabled} loading={loading} sx={sx} type={type} variant={variant}>
      {success ? <CheckIcon /> : children}
    </Button>
  );
};

export default LoadingButtonWithResult;
