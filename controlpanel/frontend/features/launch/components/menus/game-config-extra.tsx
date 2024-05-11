import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';

import {Delete as DeleteIcon} from '@mui/icons-material';
import {Button, Collapse, IconButton, List, ListItem, ListItemText, Stack, Typography} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';
import ServerPropsDialog from '../server-props-dialog';

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const serverProps = config.serverPropOverride
    ? Object.keys(config.serverPropOverride!).map((k) => ({key: k, value: config.serverPropOverride![k]}))
    : [];

  const [serverPropsDialogOpen, setServerPropsDialogOpen] = useState(false);

  const addServerProps = (key: string, value: string) => {
    const newProps = [...serverProps.filter((item) => item.key !== key), {key, value}];

    (async () => {
      updateConfig({
        serverPropOverride: newProps.map(({key, value}) => ({[key]: value})).reduce((lhs, rhs) => Object.assign(lhs, rhs), {})
      });
    })();
  };

  const removeServerProps = (key: string) => {
    const newProps = serverProps.filter((item) => item.key !== key);

    (async () => {
      updateConfig({
        serverPropOverride: newProps.map(({key, value}) => ({[key]: value})).reduce((lhs, rhs) => Object.assign(lhs, rhs), {})
      });
    })();
  };

  return {
    title: t('config_game_extra'),
    ui: (
      <Stack>
        <Typography variant="h6">{t('additional_server_properties')}</Typography>
        <Button onClick={() => setServerPropsDialogOpen(true)} variant="outlined">
          {t('additional_server_properties_add')}
        </Button>
        <List>
          <TransitionGroup>
            {serverProps.map(({key, value}) => (
              <Collapse key={key}>
                <ListItem
                  secondaryAction={
                    <IconButton
                      edge="end"
                      onClick={() => {
                        removeServerProps(key);
                      }}
                    >
                      <DeleteIcon />
                    </IconButton>
                  }
                >
                  <ListItemText primary={key} secondary={value} />
                </ListItem>
              </Collapse>
            ))}
          </TransitionGroup>
        </List>
        <ServerPropsDialog add={addServerProps} onClose={() => setServerPropsDialogOpen(false)} open={serverPropsDialogOpen} />
      </Stack>
    ),
    detail: t('value_count_label', {count: config.serverPropOverride ? Object.keys(config.serverPropOverride).length : 0}),
    variant: 'dialog',
    cancellable: true
  };
};
