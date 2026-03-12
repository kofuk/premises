import {ContentCopy as ContentCopyIcon} from '@mui/icons-material';
import {Divider, IconButton, ListItem, ListItemText, Tooltip} from '@mui/material';
import {useTranslation} from 'react-i18next';

type Prop = {
  title: string;
  children: string | React.ReactNode;
};

const CopyableListItem = ({title, children}: Prop) => {
  const [t] = useTranslation();

  const handleCopy = () => {
    navigator.clipboard.writeText(children as string);
  };

  const listItemProps = {} as any;
  if (typeof children === 'string') {
    listItemProps.secondaryAction = (
      <Tooltip title={t('copy')}>
        <IconButton aria-label="copy" edge="end" onClick={handleCopy}>
          <ContentCopyIcon />
        </IconButton>
      </Tooltip>
    );
  }

  return (
    <>
      <ListItem {...listItemProps}>{typeof children === 'string' ? <ListItemText primary={title} secondary={children} /> : children}</ListItem>
      <Divider component="li" />
    </>
  );
};

export default CopyableListItem;
