import {Typography} from '@mui/material';

type Prop = {
  message: string;
  progress: number;
};

const StatusLabel = ({message, progress}: Prop) => {
  return (
    <Typography
      component="div"
      sx={{
        background: `linear-gradient(90deg, #6aa5eb ${progress}%, #93c0f5 ${progress}%)`,
        color: 'black',
        width: 500,
        p: '5px 30px',
        borderRadius: 1000,
        border: 'solid 1px #99c1f0'
      }}
    >
      {message}
    </Typography>
  );
};

export default StatusLabel;
