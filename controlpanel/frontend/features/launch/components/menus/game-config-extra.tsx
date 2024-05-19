import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';

import {Add as AddIcon, Delete as DeleteIcon, InfoOutlined as InfoIcon} from '@mui/icons-material';
import {
  Box,
  Collapse,
  Dialog,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  ListSubheader,
  Stack,
  Switch,
  Typography
} from '@mui/material';

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

  const [motdDialogOpen, setMotdDialogOpen] = useState(false);
  const [inactiveTimeoutDialogOpen, setInactiveTimeoutDialogOpen] = useState(false);
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
        <List>
          <ListItem>
            <ListItemButton disableGutters onClick={() => setMotdDialogOpen(true)}>
              <ListItemText primary={t('server_description')} secondary={motd || <em>{t('value_not_set')}</em>} />
            </ListItemButton>
          </ListItem>
          <ListItem
            secondaryAction={
              <Switch
                checked={inactiveTimeout >= 0}
                onChange={(e) => {
                  setTimeoutMinutes(e.target.checked ? '30' : '-1');
                }}
              />
            }
          >
            <ListItemButton disableGutters onClick={() => inactiveTimeout >= 0 && setInactiveTimeoutDialogOpen(true)}>
              <ListItemText
                primary={t('inactive_timeout')}
                secondary={inactiveTimeout < 0 ? t('disabled') : t('minutes', {minutes: inactiveTimeout})}
              />
            </ListItemButton>
          </ListItem>

          <ListSubheader>{t('additional_server_properties')}</ListSubheader>
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
                  <ListItemText inset primary={key} secondary={value} />
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

        <Dialog onClose={() => setMotdDialogOpen(false)} open={motdDialogOpen}>
          <DialogTitle>{t('server_description')}</DialogTitle>
          <DialogContent sx={{mb: 1}}>
            <Box sx={{mt: 1}}>
              <SaveInput
                fullWidth
                initValue={motd}
                label={t('server_description')}
                onSave={(value) => {
                  setDescription(value);
                  setMotdDialogOpen(false);
                }}
                type="text"
              />
            </Box>
          </DialogContent>
        </Dialog>

        <Dialog onClose={() => setInactiveTimeoutDialogOpen(false)} open={inactiveTimeoutDialogOpen}>
          <DialogTitle>{t('inactive_timeout')}</DialogTitle>
          <DialogContent sx={{mb: 1}}>
            <Box sx={{mt: 1}}>
              <SaveInput
                fullWidth
                initValue={inactiveTimeout.toString()}
                label={t('inactive_timeout_input_label')}
                onSave={(value) => {
                  setTimeoutMinutes(value);
                  setInactiveTimeoutDialogOpen(false);
                }}
                type="number"
              />
            </Box>
          </DialogContent>
        </Dialog>

        <ServerPropsDialog add={addServerProps} onClose={() => setServerPropsDialogOpen(false)} open={serverPropsDialogOpen} />
      </Stack>
    ),
    detail: t('value_count_label', {count: config.serverPropOverride ? Object.keys(config.serverPropOverride).length : 0}),
    variant: 'page'
  };
};
