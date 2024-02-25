import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Box, Divider, Link, List, ListItem, ListItemButton, ListItemIcon, ListItemText} from '@mui/material';

export type MenuItem = {
  title: string;
  icon: React.ReactNode;
  detail?: React.ReactNode;
  ui: React.ReactNode;
};

export type Props = {
  items: MenuItem[];
};

const MenuContainer = ({items}: Props) => {
  const [t] = useTranslation();

  const [selectedItem, setSelectedItem] = useState(-1);

  const createMenu = () => {
    const itemElements = items.map((e, i) => (
      <>
        <ListItem key={e.title} disablePadding>
          <ListItemButton onClick={() => setSelectedItem(i)}>
            <ListItemIcon>{e.icon}</ListItemIcon>
            <ListItemText primary={e.title} secondary={e.detail} />
          </ListItemButton>
        </ListItem>
        <Divider component="li" />
      </>
    ));

    return <List>{itemElements}</List>;
  };

  if (selectedItem < 0) {
    return createMenu();
  }

  return (
    <Box>
      <Box sx={{mb: 3}}>
        <Link
          component="button"
          onClick={() => {
            setSelectedItem(-1);
          }}
        >
          {t('back')}
        </Link>
      </Box>
      {items[selectedItem].ui}
    </Box>
  );
};

export default MenuContainer;
