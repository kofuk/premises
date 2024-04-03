import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Box, Button, FormControl, FormControlLabel, FormGroup, InputLabel, MenuItem, Select, Switch} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';

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

  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);

  const {updateConfig} = useLaunchConfig();

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({
        serverVersion,
        guessServerVersion: preferDetect
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
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_server_version')}>
      {(isLoading && <Loading compact />) || (
        <>
          <Box sx={{mb: 3}}>
            <FormControlLabel
              control={<Switch checked={preferDetect} onChange={(e) => setPreferDetect(e.target.checked)} />}
              label={t('version_detect')}
            />
          </Box>

          <FormControl fullWidth>
            <InputLabel id="mc-version-select-label">{t('config_server_version')}</InputLabel>
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
      <Box sx={{textAlign: 'end'}}>
        <Button endIcon={<NextIcon />} onClick={saveAndContinue} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default ServerVersion;
