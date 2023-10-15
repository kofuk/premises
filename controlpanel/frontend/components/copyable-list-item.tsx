import React from 'react';

import {MdContentCopy} from '@react-icons/all-files/md/MdContentCopy';
import {useTranslation} from 'react-i18next';

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
            <IconButton edge="end" aria-label="copy" onClick={handleCopy}>
              <MdContentCopy />
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
