import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowBack as ArrowBackIcon} from '@mui/icons-material';

import ChooseBackup from '@/features/launch/config-item/choose-backup';
import ConfigureWorld, {LevelType} from '@/features/launch/config-item/configure-world';
import ServerVersion from '@/features/launch/config-item/server-version';
import WorldName from '@/features/launch/config-item/world-name';
import WorldSource, {WorldLocation} from '@/features/launch/config-item/world-source';

type Prop = {
  backToMenu: () => void;
  showError: (message: string) => void;
};

const ReconfigureMenu = (props: Prop) => {
  const [t] = useTranslation();

  const {backToMenu, showError} = props;

  const [serverVersion, setServerVersion] = useState('');
  const [worldSource, setWorldSource] = useState(WorldLocation.Backups);
  const [worldName, setWorldName] = useState('');
  const [backupGeneration, setBackupGeneration] = useState('');
  const [useCachedWorld, setUseCachedWorld] = useState(true);
  const [seed, setSeed] = useState('');
  const [levelType, setLevelType] = useState(LevelType.Default);
  const [currentStep, setCurrentStep] = useState(0);

  const handleStart = () => {
    (async () => {
      const data = new URLSearchParams();
      data.append('server-version', serverVersion);
      data.append('world-source', worldSource);
      if (worldSource === WorldLocation.Backups) {
        data.append('world-name', worldName);
        data.append('backup-generation', backupGeneration);
        data.append('use-cache', useCachedWorld.toString());
      } else {
        data.append('world-name', worldName);
        data.append('seed', seed);
        data.append('level-type', levelType);
      }

      try {
        const result = await fetch('/api/reconfigure', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          body: data.toString()
        }).then((resp) => resp.json());
        if (result['success']) {
          backToMenu();
        } else {
          showError(result['reason']);
        }
      } catch (err) {
        console.error(err);
      }
    })();
  };

  const xSetWorldSource = (worldSource: WorldLocation) => {
    setWorldSource(worldSource);
    if (worldSource !== WorldLocation.Backups) {
      setWorldName(worldName);
    }
  };

  const handleRequestFocus = (step: number) => {
    if (step < currentStep) {
      setCurrentStep(step);
    }
  };

  const handleNextStep = () => {
    if (currentStep < stepCount) {
      setCurrentStep(currentStep + 1);
    }
  };

  const configItems = [];
  {
    const stepIndex = configItems.length;
    configItems.push(
      <ServerVersion
        key="serverVersion"
        isFocused={currentStep === stepIndex}
        nextStep={handleNextStep}
        requestFocus={() => handleRequestFocus(stepIndex)}
        stepNum={stepIndex + 1}
        serverVersion={serverVersion}
        setServerVersion={setServerVersion}
      />
    );
  }
  {
    const stepIndex = configItems.length;
    configItems.push(
      <WorldSource
        key="worldSource"
        isFocused={currentStep === stepIndex}
        nextStep={handleNextStep}
        requestFocus={() => handleRequestFocus(stepIndex)}
        stepNum={stepIndex + 1}
        worldSource={worldSource}
        setWorldSource={xSetWorldSource}
      />
    );
  }

  if (worldSource === WorldLocation.Backups) {
    {
      const stepIndex = configItems.length;
      configItems.push(
        <ChooseBackup
          key="chooseBackup"
          isFocused={currentStep === stepIndex}
          nextStep={handleNextStep}
          requestFocus={() => handleRequestFocus(stepIndex)}
          stepNum={stepIndex + 1}
          worldName={worldName}
          backupGeneration={backupGeneration}
          useCachedWorld={useCachedWorld}
          setWorldName={setWorldName}
          setBackupGeneration={setBackupGeneration}
          setUseCachedWorld={setUseCachedWorld}
        />
      );
    }
  } else {
    {
      const stepIndex = configItems.length;
      configItems.push(
        <WorldName
          key="worldName"
          isFocused={currentStep === stepIndex}
          nextStep={handleNextStep}
          requestFocus={() => handleRequestFocus(stepIndex)}
          stepNum={stepIndex + 1}
          worldName={worldName}
          setWorldName={setWorldName}
        />
      );
    }
    {
      const stepIndex = configItems.length;
      configItems.push(
        <ConfigureWorld
          key="configureWorld"
          isFocused={currentStep === stepIndex}
          nextStep={handleNextStep}
          requestFocus={() => handleRequestFocus(stepIndex)}
          stepNum={stepIndex + 1}
          levelType={levelType}
          seed={seed}
          setLevelType={setLevelType}
          setSeed={setSeed}
        />
      );
    }
  }

  const stepCount = configItems.length;

  return (
    <div className="m-2">
      <button className="btn btn-outline-primary" onClick={props.backToMenu}>
        <ArrowBackIcon /> {t('back')}
      </button>
      <div className="m-2">
        {configItems}
        <div className="d-md-block mt-3 text-end">
          <button className="btn btn-primary bg-gradient" type="button" onClick={handleStart} disabled={currentStep !== stepCount}>
            {t('relaunch_server')}
          </button>
        </div>
      </div>
    </div>
  );
};

export default ReconfigureMenu;
