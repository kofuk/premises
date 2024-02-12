import React from 'react';

import {useTranslation} from 'react-i18next';

import {Card, Stack, Typography} from '@mui/material';

import {CodeBlock} from '@/components';
import {useRunnerStatus} from '@/utils/runner-status';

const ManualSetupPage = () => {
  const {
    extra: {textData: authCode}
  } = useRunnerStatus();

  const getInstallScriptPath = (): string => {
    const url = new URL(location.href);
    const query = url.protocol === 'https:' ? '?p=s' : '';
    return `${url.protocol}//${url.host}/_runner/install${query}`;
  };

  const [t] = useTranslation();

  return (
    <Card sx={{p: 6, mt: 12}}>
      <Stack spacing={3}>
        <Typography component="div" variant="body1">
          {t('manual_setup_summary')}
        </Typography>

        <Typography component="div" variant="body1">
          {t('manual_setup_execute_command')}
          <CodeBlock rootShell>{`curl -s '${getInstallScriptPath()}' | bash`}</CodeBlock>
        </Typography>

        <Typography component="div" variant="body1">
          {t('manual_setup_auth_code')}
          <CodeBlock>{authCode}</CodeBlock>
        </Typography>
      </Stack>
    </Card>
  );
};

export default ManualSetupPage;
