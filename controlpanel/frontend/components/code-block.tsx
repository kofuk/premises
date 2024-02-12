import React from 'react';

import styled from '@emotion/styled';

import {ContentCopy as CopyIcon} from '@mui/icons-material';
import {IconButton} from '@mui/material';
import {Box} from '@mui/system';

type Props = {
  children: string;
  rootShell?: boolean;
};

const MarginlessPre = styled.pre`
  margin: 0;
`;

const SourceBlock = ({children, rootShell}: Props) => {
  const handleCopy = () => {
    navigator.clipboard.writeText(children);
  };

  return (
    <Box sx={{position: 'relative', m: 1, p: 1, borderRadius: 1, background: '#15151a', color: '#eee'}}>
      <IconButton onClick={handleCopy} size="small" sx={{position: 'absolute', right: 0, top: 0}} type="button">
        <CopyIcon fontSize="inherit" sx={{color: '#eee'}} />
      </IconButton>
      <MarginlessPre>
        <code>
          {rootShell && '# '}
          {children}
        </code>
      </MarginlessPre>
    </Box>
  );
};

export default SourceBlock;
