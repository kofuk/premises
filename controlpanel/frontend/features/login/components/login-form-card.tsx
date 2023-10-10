import {ReactNode} from 'react';
import {Box, Card, Typography, CardContent} from '@mui/material';
import '@/i18n';

export default ({title, children}: {title: string; children: ReactNode}) => {
  return (
    <Box display="flex" justifyContent="center">
      <Card sx={{minWidth: 350, p: 3, mt: 5}}>
        <CardContent>
          <Typography variant="h4" component="h1" sx={{mb: 3}}>
            {title}
          </Typography>
          {children}
        </CardContent>
      </Card>
    </Box>
  );
};
