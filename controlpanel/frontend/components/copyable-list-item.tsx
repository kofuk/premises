import React from 'react';

import {useTranslation} from 'react-i18next';

import {ContentCopy as ContentCopyIcon} from '@mui/icons-material';
import {Divider, IconButton, ListItem, ListItemText, Tooltip} from '@mui/material';

type Prop = {
  title: string;
  children: string;
};

const CopyableListItem = ({title, children}: Prop) => {
  const [t] = useTranslation();

  const handleCopy = () => {
    navigator.clipboard.writeText(children);
  };

  return (
    <>
      <ListItem
        secondaryAction={
          <Tooltip title={t('copy')}>
            <IconButton aria-label="copy" edge="end" onClick={handleCopy}>
              <ContentCopyIcon />
            </IconButton>
          </Tooltip>
        }
      >
        <ListItemText primary={title} secondary={children} />
      </ListItem>
      <Divider />
    </>
  );
};

export default CopyableListItem;
