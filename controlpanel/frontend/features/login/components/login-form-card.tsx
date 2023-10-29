import React, {ReactNode} from 'react';

import {Box, Card, CardContent, Typography} from '@mui/material';

const LoginFormCard = ({title, children}: {title: string; children: ReactNode}) => {
  return (
    <Box display="flex" justifyContent="center">
      <Card sx={{minWidth: 350, p: 3, mt: 5}}>
        <CardContent>
          <Typography component="h1" sx={{mb: 3}} variant="h4">
            {title}
          </Typography>
          {children}
        </CardContent>
      </Card>
    </Box>
  );
};

export default LoginFormCard;
