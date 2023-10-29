import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

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

const ServerVersion = ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  serverVersion,
  setServerVersion
}: ItemProp & {
  serverVersion: string;
  setServerVersion: (val: string) => void;
}) => {
  const [t] = useTranslation();

  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);

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
        <option key={e.name} value={e.name}>
          {e.name}
        </option>
      ));
  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_server_version')}>
      {(isLoading && <Loading compact />) || (
        <>
          <select
            aria-label={t('config_server_version')}
            className="form-select"
            onChange={(e) => handleChange(e.target.value)}
            value={serverVersion}
          >
            {versions}
          </select>
          <div className="m-1 form-check form-switch">
            <input
              checked={showStable}
              className="form-check-input"
              id="showStable"
              onChange={() => {
                setShowStable(!showStable);
                postUpdateCondition(mcVersions!);
              }}
              type="checkbox"
            />
            <label className="form-check-label" htmlFor="showStable">
              {t('version_show_stable')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              checked={showSnapshot}
              className="form-check-input"
              id="showSnapshot"
              onChange={() => {
                setShowSnapshot(!showSnapshot);
                postUpdateCondition(mcVersions!);
              }}
              type="checkbox"
            />
            <label className="form-check-label" htmlFor="showSnapshot">
              {t('version_show_snapshot')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              checked={showBeta}
              className="form-check-input"
              id="showSnapshot"
              onChange={() => {
                setShowBeta(!showBeta);
                postUpdateCondition(mcVersions!);
              }}
              type="checkbox"
            />
            <label className="form-check-label" htmlFor="showBeta">
              {t('version_show_beta')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              checked={showAlpha}
              className="form-check-input"
              id="showSnapshot"
              onChange={() => {
                setShowAlpha(!showAlpha);
                postUpdateCondition(mcVersions!);
              }}
              type="checkbox"
            />
            <label className="form-check-label" htmlFor="showAlpha">
              {t('version_show_alpha')}
            </label>
          </div>
        </>
      )}
      <div className="m-1 text-end">
        <button className="btn btn-primary" disabled={isLoading} onClick={nextStep} type="button">
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default ServerVersion;
