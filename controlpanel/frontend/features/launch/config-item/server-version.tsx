import React, {useEffect, useState} from 'react';

import {useTranslation} from 'react-i18next';

import {useMCVersions} from '@/api';
import Loading from '@/components/loading';
import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

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
        <option value={e.name} key={e.name}>
          {e.name}
        </option>
      ));
  return (
    <ConfigContainer title={t('config_server_version')} isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum}>
      {(isLoading && <Loading compact />) || (
        <>
          <select
            className="form-select"
            aria-label={t('config_server_version')}
            value={serverVersion}
            onChange={(e) => handleChange(e.target.value)}
          >
            {versions}
          </select>
          <div className="m-1 form-check form-switch">
            <input
              className="form-check-input"
              type="checkbox"
              id="showStable"
              checked={showStable}
              onChange={() => {
                setShowStable(!showStable);
                postUpdateCondition(mcVersions!);
              }}
            />
            <label className="form-check-label" htmlFor="showStable">
              {t('version_show_stable')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              className="form-check-input"
              type="checkbox"
              id="showSnapshot"
              checked={showSnapshot}
              onChange={() => {
                setShowSnapshot(!showSnapshot);
                postUpdateCondition(mcVersions!);
              }}
            />
            <label className="form-check-label" htmlFor="showSnapshot">
              {t('version_show_snapshot')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              className="form-check-input"
              type="checkbox"
              id="showSnapshot"
              checked={showBeta}
              onChange={() => {
                setShowBeta(!showBeta);
                postUpdateCondition(mcVersions!);
              }}
            />
            <label className="form-check-label" htmlFor="showBeta">
              {t('version_show_beta')}
            </label>
          </div>
          <div className="m-1 form-check form-switch">
            <input
              className="form-check-input"
              type="checkbox"
              id="showSnapshot"
              checked={showAlpha}
              onChange={() => {
                setShowAlpha(!showAlpha);
                postUpdateCondition(mcVersions!);
              }}
            />
            <label className="form-check-label" htmlFor="showAlpha">
              {t('version_show_alpha')}
            </label>
          </div>
        </>
      )}
      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep} disabled={isLoading}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default ServerVersion;
