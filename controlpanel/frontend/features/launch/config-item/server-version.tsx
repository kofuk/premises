import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';
import {TransitionGroup} from 'react-transition-group';

import {Delete as DeleteIcon, ExpandMore as ExpandMoreIcon, ArrowDownward as NextIcon} from '@mui/icons-material';
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Collapse,
  FormControl,
  FormControlLabel,
  FormGroup,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemText,
  MenuItem,
  Select,
  Stack,
  Switch,
  Typography
} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';
import ServerPropsDialog from '../components/server-props-dialog';

import ConfigContainer from './config-container';
import {ItemProp} from './prop';

import {useMCVersions} from '@/api';
import {Loading} from '@/components';

type McVersion = {
  name: string;
  isStable: boolean;
  channel: string;
  releaseDate: string;
};

const ServerVersion = ({isFocused, nextStep, requestFocus, stepNum}: ItemProp) => {
  const [t] = useTranslation();

  const [serverVersion, setServerVersion] = useState('');
  const [preferDetect, setPreferDetect] = useState(true);
  const [serverProps, setServerProps] = useState<{key: string; value: string}[]>([]);
  const [serverPropsDialogOpen, setServerPropsDialogOpen] = useState(false);

  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);

  const {updateConfig} = useLaunchConfig();

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({
        serverVersion,
        guessServerVersion: preferDetect,
        serverPropOverride: serverProps.map(({key, value}) => ({[key]: value})).reduce((lhs, rhs) => Object.assign(lhs, rhs), {})
      });
      nextStep();
    })();
  };

  const {data: mcVersions, isLoading} = useMCVersions();
  useEffect(() => {
    if (mcVersions) {
      postUpdateCondition(mcVersions);
    }
  }, [mcVersions]);

  const handleChange = (val: string) => {
    setServerVersion(val);
  };

  const handleAddServerProps = (key: string, value: string) => {
    setServerProps((current) => [...current.filter((item) => item.key !== key), {key, value}]);
  };

  const postUpdateCondition = (versionsData: McVersion[]) => {
    const versions = versionsData
      .filter((e) => showStable || e.channel !== 'stable')
      .filter((e) => showSnapshot || e.channel !== 'snapshot')
      .filter((e) => showBeta || e.channel !== 'beta')
      .filter((e) => showAlpha || e.channel !== 'alpha');
    if (!versions.find((e) => e.name === serverVersion)) {
      if (versions.length > 0) {
        setServerVersion(versions[0].name);
      } else if (mcVersions!.length > 0) {
        setServerVersion(versionsData[0].name);
      }
    }
  };

  const versions =
    mcVersions &&
    mcVersions
      .filter((e) => showStable || e.channel !== 'stable')
      .filter((e) => showSnapshot || e.channel !== 'snapshot')
      .filter((e) => showBeta || e.channel !== 'beta')
      .filter((e) => showAlpha || e.channel !== 'alpha')
      .map((e) => (
        <MenuItem key={e.name} value={e.name}>
          {e.name}
        </MenuItem>
      ));
  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_server_settings')}>
      {(isLoading && <Loading compact />) || (
        <>
          <Box sx={{mb: 3}}>
            <FormControlLabel
              control={<Switch checked={preferDetect} onChange={(e) => setPreferDetect(e.target.checked)} />}
              label={t('version_detect')}
            />
          </Box>

          <FormControl fullWidth>
            <InputLabel id="mc-version-select-label">{t('server_version')}</InputLabel>
            <Select
              label={t('config_server_version')}
              labelId="mc-version-select-label"
              onChange={(e) => handleChange(e.target.value)}
              value={serverVersion}
            >
              {versions}
            </Select>
          </FormControl>
          <FormGroup row>
            <FormControlLabel
              control={
                <Switch
                  checked={showStable}
                  onChange={() => {
                    setShowStable(!showStable);
                    postUpdateCondition(mcVersions!);
                  }}
                  size="small"
                />
              }
              label={t('version_show_stable')}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={showSnapshot}
                  onChange={() => {
                    setShowSnapshot(!showSnapshot);
                    postUpdateCondition(mcVersions!);
                  }}
                  size="small"
                />
              }
              label={t('version_show_snapshot')}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={showBeta}
                  onChange={() => {
                    setShowBeta(!showBeta);
                    postUpdateCondition(mcVersions!);
                  }}
                  size="small"
                />
              }
              label={t('version_show_beta')}
            />
            <FormControlLabel
              control={
                <Switch
                  checked={showAlpha}
                  onChange={() => {
                    setShowAlpha(!showAlpha);
                    postUpdateCondition(mcVersions!);
                  }}
                  size="small"
                />
              }
              label={t('version_show_alpha')}
            />
          </FormGroup>
        </>
      )}
      <Accordion sx={{my: 1}}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>{t('advanced_server_settings')}</AccordionSummary>
        <AccordionDetails>
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
                            setServerProps((prev) => prev.filter((item) => item.key !== key));
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
          </Stack>
        </AccordionDetails>
      </Accordion>

      <ServerPropsDialog add={handleAddServerProps} onClose={() => setServerPropsDialogOpen(false)} open={serverPropsDialogOpen} />

      <Box sx={{textAlign: 'end'}}>
        <Button endIcon={<NextIcon />} onClick={saveAndContinue} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default ServerVersion;
