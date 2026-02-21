import {ArrowBack as BackIcon, Close as CloseIcon} from '@mui/icons-material';
import {
  Box,
  Dialog,
  DialogContent,
  DialogTitle,
  Divider,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Typography
} from '@mui/material';
import {Fragment, useState} from 'react';

export type MenuItem = {
  title: string;
  icon?: React.ReactNode;
  detail?: React.ReactNode;
  ui: React.ReactNode;
  disabled?: boolean;
} & (
  | {
      variant?: 'page';
    }
  | {
      variant: 'dialog';
      cancellable?: boolean;
    }
);

export type Props = {
  items: MenuItem[];
  menuFooter?: React.ReactNode;
};

const MenuContainer = ({items, menuFooter}: Props) => {
  const [selectedItem, setSelectedItem] = useState(-1);

  const backToMenu = () => {
    setSelectedItem(-1);
  };

  const createMenu = () => {
    const itemElements = items
      .filter((e) => !e.disabled)
      .map((e, i) => (
        <Fragment key={e.title}>
          <ListItem disablePadding>
            <ListItemButton onClick={() => setSelectedItem(i)}>
              {e.icon && <ListItemIcon>{e.icon}</ListItemIcon>}
              <ListItemText primary={e.title} secondary={e.detail} />
            </ListItemButton>
          </ListItem>
          <Divider component="li" />
        </Fragment>
      ));

    return <List>{itemElements}</List>;
  };

  const dialogs = items.map((e, i) => {
    if (e.disabled || e.variant !== 'dialog') {
      return null;
    }
    const dialogProps = e.cancellable
      ? {
          onClose: backToMenu
        }
      : {};
    return (
      <Fragment key={e.title}>
        <Dialog fullWidth open={i === selectedItem} scroll="paper" {...dialogProps}>
          <DialogTitle>
            {e.cancellable && (
              <IconButton onClick={backToMenu}>
                <CloseIcon />
              </IconButton>
            )}
            {e.title}
          </DialogTitle>
          <DialogContent>{e.ui}</DialogContent>
        </Dialog>
      </Fragment>
    );
  });

  if (selectedItem < 0 || items[selectedItem].variant === 'dialog') {
    return (
      <Box>
        {createMenu()}
        {menuFooter}
        {dialogs}
      </Box>
    );
  }

  return (
    <Box>
      <Typography sx={{textAlign: 'middle'}} variant="h5">
        <IconButton onClick={backToMenu}>
          <BackIcon />
        </IconButton>
        {items[selectedItem].title}
      </Typography>
      {items[selectedItem].ui}
    </Box>
  );
};

export default MenuContainer;
