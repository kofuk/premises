import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Box, FormControl, FormControlLabel, FormGroup, InputLabel, MenuItem as MUIMenuItem, Select, Switch} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {valueLabel} from './common';

import {useMCVersions} from '@/api';
import {Loading} from '@/components';

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const [serverVersion, setServerVersion] = useState(config.serverVersion || '');
  const [guessServerVersion, setGuessServerVersion] = useState(config.guessServerVersion === undefined ? true : config.guessServerVersion);

  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);

  const {data: mcVersions, isLoading} = useMCVersions();
  const versions =
    mcVersions &&
    mcVersions
      .filter((e) => showStable || e.channel !== 'stable')
      .filter((e) => showSnapshot || e.channel !== 'snapshot')
      .filter((e) => showBeta || e.channel !== 'beta')
      .filter((e) => showAlpha || e.channel !== 'alpha')
      .map((e) => (
        <MUIMenuItem key={e.name} value={e.name}>
          {e.name}
        </MUIMenuItem>
      ));

  return {
    title: t('config_server_settings'),
    ui: isLoading ? (
      <Loading compact />
    ) : (
      <>
        <Box sx={{mb: 3}}>
          <FormControlLabel
            control={<Switch checked={guessServerVersion} onChange={(e) => setGuessServerVersion(e.target.checked)} />}
            label={t('version_detect')}
          />
        </Box>

        <FormControl fullWidth>
          <InputLabel id="mc-version-select-label">{t('server_version')}</InputLabel>
          <Select
            label={t('config_server_version')}
            labelId="mc-version-select-label"
            onChange={(e) => setServerVersion(e.target.value)}
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
                  if (showStable) {
                    setServerVersion('');
                  }
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
                  if (showSnapshot) {
                    setServerVersion('');
                  }
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
                  if (showBeta) {
                    setServerVersion('');
                  }
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
                  if (showAlpha) {
                    setServerVersion('');
                  }
                }}
                size="small"
              />
            }
            label={t('version_show_alpha')}
          />
        </FormGroup>
      </>
    ),
    detail: valueLabel(config.serverVersion),
    variant: 'dialog',
    cancellable: true,
    action: {
      label: t('save'),
      callback: () => {
        updateConfig({
          serverVersion,
          guessServerVersion
        });
      }
    }
  };
};
