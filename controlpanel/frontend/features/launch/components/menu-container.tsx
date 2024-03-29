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
  menuFooter?: React.ReactNode;
};

const MenuContainer = ({items, menuFooter}: Props) => {
  const [t] = useTranslation();

  const [selectedItem, setSelectedItem] = useState(-1);

  const createMenu = () => {
    const itemElements = items.map((e, i) => (
      <React.Fragment key={e.title}>
        <ListItem disablePadding>
          <ListItemButton onClick={() => setSelectedItem(i)}>
            <ListItemIcon>{e.icon}</ListItemIcon>
            <ListItemText primary={e.title} secondary={e.detail} />
          </ListItemButton>
        </ListItem>
        <Divider component="li" />
      </React.Fragment>
    ));

    return <List>{itemElements}</List>;
  };

  if (selectedItem < 0) {
    return (
      <Box>
        {createMenu()}
        {menuFooter}
      </Box>
    );
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
