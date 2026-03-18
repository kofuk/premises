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
  Switch,
  Tooltip
} from '@mui/material';
import {useState} from 'react';
import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';
import SaveInput from '@/components/save-input';
import {useLaunchConfig} from '../launch-config';
import type {MenuItem} from '../menu-container';
import ServerPropsDialog from '../server-props-dialog';

enum OpenedDialog {
  NONE,
  MOTD,
  INACTIVE_TIMEOUT,
  O11Y,
  SERVER_PROPS
}

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const serverProps = config.serverPropOverride
    ? Object.keys(config.serverPropOverride!).map((k) => ({key: k, value: config.serverPropOverride![k]}))
    : [];
  const motd = config.motd || '';
  const inactiveTimeout = config.inactiveTimeout || -1;
  const otlpEndpoint = config.otlpEndpoint || '';
  const metricExportIntervalSec = config.metricExportIntervalSec || 10;

  const [openedDialog, setOpenedDialog] = useState(OpenedDialog.NONE);

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

  const setOtlpEndpoint = (otlpEndpoint: string) => {
    if (otlpEndpoint.match(/^https?:\/\/[-a-zA-Z0-9.]:[0-9]+/)) {
      updateConfig({otlpEndpoint: otlpEndpoint});
    }
  };

  const setMetricExportIntervalSec = (intervalSec: string) => {
    updateConfig({metricExportIntervalSec: parseInt(intervalSec, 10)});
  };

  return {
    title: t('launch.server_extra'),
    ui: (
      <Box>
        <List>
          <ListItem>
            <ListItemButton disableGutters onClick={() => setOpenedDialog(OpenedDialog.MOTD)}>
              <ListItemText primary={t('launch.server_extra.motd')} secondary={motd || <em>{t('launch.server_extra.motd.not_set')}</em>} />
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
            <ListItemButton disableGutters onClick={() => inactiveTimeout >= 0 && setOpenedDialog(OpenedDialog.INACTIVE_TIMEOUT)}>
              <ListItemText
                primary={
                  <>
                    {t('launch.server_extra.inactive_timeout')}
                    <Tooltip title={t('launch.server_extra.inactive_timeout.notice')}>
                      <InfoIcon sx={{opacity: 0.6}} />
                    </Tooltip>
                  </>
                }
                secondary={
                  inactiveTimeout < 0
                    ? t('launch.server_extra.inactive_timeout.disabled')
                    : t('launch.server_extra.inactive_timeout.minutes', {minutes: inactiveTimeout})
                }
              />
            </ListItemButton>
          </ListItem>

          <ListItem>
            <ListItemButton disableGutters onClick={() => setOpenedDialog(OpenedDialog.O11Y)}>
              <ListItemText primary={t('launch.server_extra.o11y')} secondary={otlpEndpoint || <em>{t('launch.server_extra.o11y.not_set')}</em>} />
            </ListItemButton>
          </ListItem>

          <ListSubheader disableSticky>
            {t('launch.server_extra.server_properties')}
            <Tooltip title={t('launch.server_extra.server_properties.notice')}>
              <InfoIcon sx={{opacity: 0.6}} />
            </Tooltip>
          </ListSubheader>
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
            <ListItemButton onClick={() => setOpenedDialog(OpenedDialog.SERVER_PROPS)}>
              <ListItemIcon>
                <AddIcon />
              </ListItemIcon>
              {t('launch.server_extra.server_properties.add')}
            </ListItemButton>
          </ListItem>
        </List>

        <Dialog onClose={() => setOpenedDialog(OpenedDialog.NONE)} open={openedDialog === OpenedDialog.MOTD}>
          <DialogTitle>{t('launch.server_extra.motd')}</DialogTitle>
          <DialogContent sx={{mb: 1}}>
            <Box sx={{mt: 1}}>
              <SaveInput
                fullWidth
                initValue={motd}
                label={t('launch.server_extra.motd')}
                onSave={(value) => {
                  setDescription(value);
                  setOpenedDialog(OpenedDialog.NONE);
                }}
                type="text"
              />
            </Box>
          </DialogContent>
        </Dialog>

        <Dialog onClose={() => setOpenedDialog(OpenedDialog.NONE)} open={openedDialog === OpenedDialog.INACTIVE_TIMEOUT}>
          <DialogTitle>{t('launch.server_extra.inactive_timeout')}</DialogTitle>
          <DialogContent sx={{mb: 1}}>
            <Box sx={{mt: 1}}>
              <SaveInput
                fullWidth
                initValue={inactiveTimeout.toString()}
                label={t('launch.server_extra.inactive_timeout.input_label')}
                onSave={(value) => {
                  setTimeoutMinutes(value);
                  setOpenedDialog(OpenedDialog.NONE);
                }}
                type="number"
              />
            </Box>
          </DialogContent>
        </Dialog>

        <Dialog onClose={() => setOpenedDialog(OpenedDialog.NONE)} open={openedDialog === OpenedDialog.O11Y}>
          <DialogTitle>{t('launch.server_extra.o11y')}</DialogTitle>
          <DialogContent sx={{mb: 1}}>
            <Box sx={{mt: 1}}>
              <SaveInput
                fullWidth
                initValue={otlpEndpoint}
                label={t('launch.server_extra.o11y.otlp_endpoint.input_label')}
                onSave={(value) => {
                  setOtlpEndpoint(value);
                  setOpenedDialog(OpenedDialog.NONE);
                }}
                type="string"
              />
              <SaveInput
                fullWidth
                initValue={metricExportIntervalSec.toString()}
                label={t('launch.server_extra.o11y.metric_export_interval_sec.input_label')}
                onSave={(value) => {
                  setMetricExportIntervalSec(value);
                  setOpenedDialog(OpenedDialog.NONE);
                }}
                type="number"
              />
            </Box>
          </DialogContent>
        </Dialog>

        <ServerPropsDialog
          add={addServerProps}
          onClose={() => setOpenedDialog(OpenedDialog.NONE)}
          open={openedDialog === OpenedDialog.SERVER_PROPS}
        />
      </Box>
    ),
    variant: 'page'
  };
};
