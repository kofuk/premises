import {useState} from 'react';
import {VscDebugStart} from '@react-icons/all-files/vsc/VscDebugStart';

import '@/i18n';
import {t} from 'i18next';

import MachineType from '@/features/launch/config-item/machine-type';
import ServerVersion from '@/features/launch/config-item/server-version';
import WorldSource from '@/features/launch/config-item/world-source';
import {WorldLocation} from '@/features/launch/config-item/world-source';
import ChooseBackup from '@/features/launch/config-item/choose-backup';
import WorldName from '@/features/launch/config-item/world-name';
import ConfigureWorld from '@/features/launch/config-item/configure-world';
import {LevelType} from '@/features/launch/config-item/configure-world';

type Prop = {
  showError: (message: string) => void;
};

export default (props: Prop) => {
  const {showError} = props;

  const [machineType, setMachineType] = useState('4g');
  const [serverVersion, setServerVersion] = useState('');
  const [worldSource, setWorldSource] = useState(WorldLocation.Backups);
  const [worldName, setWorldName] = useState('');
  const [backupGeneration, setBackupGeneration] = useState('');
  const [useCachedWorld, setUseCachedWorld] = useState(true);
  const [seed, setSeed] = useState('');
  const [levelType, setLevelType] = useState(LevelType.Default);
  const [currentStep, setCurrentStep] = useState(0);

  const handleStart = () => {
    const data = new URLSearchParams();
    data.append('machine-type', machineType);
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

    fetch('/api/launch', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: data.toString()
    })
      .then((resp) => resp.json())
      .then((resp) => {
        if (!resp['success']) {
          showError(resp['message']);
        }
      });
  };

  const xSetWorldSource = (worldSource: WorldLocation) => {
    setWorldSource(worldSource);
    if (worldSource !== WorldLocation.Backups) {
      setWorldName('');
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
      <MachineType
        key="machineType"
        isFocused={currentStep === stepIndex}
        nextStep={handleNextStep}
        requestFocus={() => handleRequestFocus(stepIndex)}
        stepNum={stepIndex + 1}
        machineType={machineType}
        setMachineType={setMachineType}
      />
    );
  }
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
    <div className="my-5 card mx-auto">
      <div className="card-body">
        <form>
          {configItems}
          <div className="d-md-block mt-3 text-end">
            <button className="btn btn-primary bg-gradient" type="button" onClick={handleStart} disabled={currentStep !== stepCount}>
              <VscDebugStart /> {t('launch_server')}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};