import {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {InfoOutlined as InfoIcon} from '@mui/icons-material';
import {Box, FormControl, FormControlLabel, FormGroup, InputLabel, MenuItem as MUIMenuItem, Select, Switch, Tooltip} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {valueLabel} from './common';

import {useMCVersions} from '@/api';
import Loading from '@/components/loading';

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const serverVersion = config.serverVersion || '';
  const guessServerVersion = !!config.guessServerVersion;

  const setGuessServerVersion = (enable: boolean) => {
    updateConfig({
      guessServerVersion: enable
    });
  };

  const setServerVersion = (version: string) => {
    updateConfig({
      serverVersion: version
    });
  };

  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);

  const {data: mcVersions, isLoading} = useMCVersions();
  const versions =
    mcVersions &&
    mcVersions
      .filter(
        (e) =>
          e.name === serverVersion ||
          ((showStable || e.channel !== 'stable') &&
            (showSnapshot || e.channel !== 'snapshot') &&
            (showBeta || e.channel !== 'beta') &&
            (showAlpha || e.channel !== 'alpha'))
      )
      .map((e) => (
        <MUIMenuItem key={e.name} value={e.name}>
          {e.name}
        </MUIMenuItem>
      ));

  return {
    title: t('launch.server_version'),
    ui: isLoading ? (
      <Loading compact />
    ) : (
      <>
        <Box sx={{mb: 3}}>
          <FormControlLabel
            control={<Switch checked={guessServerVersion} onChange={(e) => setGuessServerVersion(e.target.checked)} />}
            label={
              <>
                {t('launch.server_version.guess_from_world')}
                <Tooltip sx={{opacity: 0.6}} title={t('launch.server_version.notice')}>
                  <InfoIcon />
                </Tooltip>
              </>
            }
          />
        </Box>

        <FormControl fullWidth>
          <InputLabel id="mc-version-select-label">{t('launch.server_version.version')}</InputLabel>
          <Select
            label={t('launch.server_version.version')}
            labelId="mc-version-select-label"
            onChange={(e) => setServerVersion(e.target.value)}
            value={serverVersion}
          >
            {versions}
          </Select>
        </FormControl>
        <FormGroup row>
          <FormControlLabel
            control={<Switch checked={showStable} onChange={() => setShowStable(!showStable)} size="small" />}
            label={t('launch.server_version.stable')}
          />
          <FormControlLabel
            control={<Switch checked={showSnapshot} onChange={() => setShowSnapshot(!showSnapshot)} size="small" />}
            label={t('launch.server_version.snapshot')}
          />
          <FormControlLabel
            control={<Switch checked={showBeta} onChange={() => setShowBeta(!showBeta)} size="small" />}
            label={t('launch.server_version.beta')}
          />
          <FormControlLabel
            control={<Switch checked={showAlpha} onChange={() => setShowAlpha(!showAlpha)} size="small" />}
            label={t('launch.server_version.alpha')}
          />
        </FormGroup>
      </>
    ),
    detail: valueLabel(config.serverVersion),
    variant: 'dialog',
    cancellable: true
  };
};
