import {Card, Stack, Typography} from '@mui/material';
import {useTranslation} from 'react-i18next';

import CodeBlock from '@/components/code-block';
import {useRunnerStatus} from '@/utils/runner-status';

const ManualSetupPage = () => {
  const {
    extra: {textData: authCode}
  } = useRunnerStatus();

  const getInstallScriptPath = (): string => {
    const url = new URL(location.href);
    const query = url.protocol === 'http:' ? '?s=0' : '';
    return `${url.protocol}//${url.host}/_/install${query}`;
  };

  const [t] = useTranslation();

  return (
    <Card sx={{p: 6, mt: 12}} variant="outlined">
      <Stack spacing={3}>
        <Typography component="div" variant="body1">
          {t('launch.manual_setup.summary')}
        </Typography>

        <Typography component="div" variant="body1">
          {t('launch.manual_setup.execute_command')}
          <CodeBlock rootShell>{`curl -s '${getInstallScriptPath()}' | bash`}</CodeBlock>
        </Typography>

        <Typography component="div" variant="body1">
          {t('launch.manual_setup.auth_code')}
          <CodeBlock>{authCode}</CodeBlock>
        </Typography>
      </Stack>
    </Card>
  );
};

export default ManualSetupPage;
