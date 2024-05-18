import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';

import {Add as AddIcon, Delete as DeleteIcon, InfoOutlined as InfoIcon} from '@mui/icons-material';
import {Collapse, IconButton, List, ListItem, ListItemButton, ListItemIcon, ListItemText, ListSubheader, Stack, Typography} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';
import ServerPropsDialog from '../server-props-dialog';

import SaveInput from '@/components/save-input';

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const serverProps = config.serverPropOverride
    ? Object.keys(config.serverPropOverride!).map((k) => ({key: k, value: config.serverPropOverride![k]}))
    : [];
  const motd = config.motd || '';
  const inactiveTimeout = config.inactiveTimeout || -1;

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

    updateConfig({
      serverPropOverride: newProps.map(({key, value}) => ({[key]: value})).reduce((lhs, rhs) => Object.assign(lhs, rhs), {})
    });
  };

  const setDescription = (desc: string) => {
    updateConfig({motd: desc});
  };

  const setTimeoutMinutes = (minutes: string) => {
    updateConfig({inactiveTimeout: parseInt(minutes, 10)});
  };

  return {
    title: t('config_game_extra'),
    ui: (
      <Stack spacing={2} sx={{mt: 1}}>
        <SaveInput fullWidth initValue={motd} label={t('server_description')} onSave={setDescription} type="text" />
        <SaveInput fullWidth initValue={`${inactiveTimeout}`} label={t('inactive_timeout')} onSave={setTimeoutMinutes} type="number" />

        <List subheader={<ListSubheader>{t('additional_server_properties')}</ListSubheader>}>
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
          <ListItem>
            <ListItemButton onClick={() => setServerPropsDialogOpen(true)}>
              <ListItemIcon>
                <AddIcon />
              </ListItemIcon>
              {t('additional_server_properties_add')}
            </ListItemButton>
          </ListItem>
          <Stack component="li" direction="row" gap={1} sx={{ml: 5, mb: 2, opacity: 0.9}}>
            <InfoIcon />
            <Typography variant="body1">{t('server_properties_description')}</Typography>
          </Stack>
        </List>
        <ServerPropsDialog add={addServerProps} onClose={() => setServerPropsDialogOpen(false)} open={serverPropsDialogOpen} />
      </Stack>
    ),
    detail: t('value_count_label', {count: config.serverPropOverride ? Object.keys(config.serverPropOverride).length : 0}),
    variant: 'page'
  };
};
