import {useState, useEffect} from 'react';
import {IoIosRefresh} from '@react-icons/all-files/io/IoIosRefresh';

import '../../i18n';
import {t} from 'i18next';

import {ItemProp} from './prop';
import ConfigContainer from './config-container';

type McVersion = {
  name: string;
  isStable: boolean;
  channel: string;
  releaseDate: string;
};

export default ({
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
  const [mcVersions, setMcVersions] = useState<McVersion[]>([]);
  const [showStable, setShowStable] = useState(true);
  const [showSnapshot, setShowSnapshot] = useState(false);
  const [showAlpha, setShowAlpha] = useState(false);
  const [showBeta, setShowBeta] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    fetch('/api/mcversions')
      .then((resp) => resp.json())
      .then((resp) => {
        setMcVersions(resp);
        postUpdateCondition(resp);
      });
  }, []);

  const handleRefresh = () => {
    setRefreshing(true);
    fetch('/api/mcversions?reload')
      .then((resp) => resp.json())
      .then((resp) => {
        setMcVersions(resp);
        postUpdateCondition(resp);
        setRefreshing(false);
      });
  };

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
      } else if (mcVersions.length > 0) {
        setServerVersion(versionsData[0].name);
      }
    }
  };

  const versions = mcVersions
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
      <select className="form-select" area-label={t('config_server_version')} value={serverVersion} onChange={(e) => handleChange(e.target.value)}>
        {versions}
      </select>
      <div className="m-1 text-end">
        <button type="button" className="btn btn-sm btn-outline-secondary" onClick={handleRefresh} disabled={refreshing}>
          {refreshing ? <div className="spinner-border spinner-border-sm me-1" role="status"></div> : <IoIosRefresh />}
          {t('refresh')}
        </button>
      </div>
      <div className="m-1 form-check form-switch">
        <input
          className="form-check-input"
          type="checkbox"
          id="showStable"
          checked={showStable}
          onChange={() => {
            setShowStable(!showStable);
            postUpdateCondition(mcVersions);
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
            postUpdateCondition(mcVersions);
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
            postUpdateCondition(mcVersions);
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
            postUpdateCondition(mcVersions);
          }}
        />
        <label className="form-check-label" htmlFor="showAlpha">
          {t('version_show_alpha')}
        </label>
      </div>
      <div className="m-1 text-end">
        <button type="button" className="btn btn-primary" onClick={nextStep}>
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};
