import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Divider,
  Link,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText
} from '@mui/material';

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
      action?: {
        label: string;
        callback: () => void;
      };
    }
);

export type Props = {
  items: MenuItem[];
  menuFooter?: React.ReactNode;
};

const MenuContainer = ({items, menuFooter}: Props) => {
  const [t] = useTranslation();

  const [selectedItem, setSelectedItem] = useState(-1);

  const backToMenu = () => {
    setSelectedItem(-1);
  };

  const createMenu = () => {
    const itemElements = items
      .filter((e) => !e.disabled)
      .map((e, i) => (
        <React.Fragment key={`${i}`}>
          <ListItem disablePadding>
            <ListItemButton onClick={() => setSelectedItem(i)}>
              {e.icon && <ListItemIcon>{e.icon}</ListItemIcon>}
              <ListItemText primary={e.title} secondary={e.detail} />
            </ListItemButton>
          </ListItem>
          <Divider component="li" />
        </React.Fragment>
      ));

    return <List>{itemElements}</List>;
  };

  const dialogs = items.map((e, i) => {
    if (e.disabled || e.variant !== 'dialog') {
      return;
    }
    const dialogProps = e.cancellable
      ? {
          onClose: backToMenu
        }
      : {};
    return (
      <React.Fragment key={`${i}`}>
        <Dialog fullWidth open={i == selectedItem} scroll="paper" {...dialogProps}>
          <DialogTitle>{e.title}</DialogTitle>
          <DialogContent>{e.ui}</DialogContent>
          {e.action && (
            <DialogActions>
              {e.cancellable && <Button onClick={backToMenu}>{t('cancel')}</Button>}
              <Button
                autoFocus
                onClick={() => {
                  e.action!.callback();
                  backToMenu();
                }}
              >
                {e.action.label}
              </Button>
            </DialogActions>
          )}
        </Dialog>
      </React.Fragment>
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
      <Box sx={{mb: 3}}>
        <Link component="button" onClick={backToMenu}>
          {t('back')}
        </Link>
      </Box>
      {items[selectedItem].variant !== 'dialog' && items[selectedItem].ui}
    </Box>
  );
};

export default MenuContainer;
