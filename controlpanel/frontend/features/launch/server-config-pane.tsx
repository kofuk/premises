import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {PlayArrow as StartIcon} from '@mui/icons-material';
import {Box, Button, Card} from '@mui/material';

import ChooseBackup from '@/features/launch/config-item/choose-backup';
import ConfigureWorld, {LevelType} from '@/features/launch/config-item/configure-world';
import MachineType from '@/features/launch/config-item/machine-type';
import ServerVersion from '@/features/launch/config-item/server-version';
import WorldName from '@/features/launch/config-item/world-name';
import WorldSource, {WorldLocation} from '@/features/launch/config-item/world-source';

const ServerConfigPane = () => {
  const [t] = useTranslation();

  const [machineType, setMachineType] = useState('4g');
  const [serverVersion, setServerVersion] = useState('');
  const [preferDetect, setPreferDetect] = useState(true);
  const [worldSource, setWorldSource] = useState(WorldLocation.Backups);
  const [worldName, setWorldName] = useState('');
  const [backupGeneration, setBackupGeneration] = useState('@/latest');
  const [seed, setSeed] = useState('');
  const [levelType, setLevelType] = useState(LevelType.Default);
  const [currentStep, setCurrentStep] = useState(0);

  const handleStart = () => {
    (async () => {
      const data = new URLSearchParams();
      data.append('machine-type', machineType);
      data.append('server-version', serverVersion);
      data.append('prefer-detect', preferDetect.toString());
      data.append('world-source', worldSource);
      if (worldSource === WorldLocation.Backups) {
        data.append('world-name', worldName);
        data.append('backup-generation', backupGeneration);
      } else {
        data.append('world-name', worldName);
        data.append('seed', seed);
        data.append('level-type', levelType);
      }

      try {
        const result = await fetch('/api/launch', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          body: data.toString()
        }).then((resp) => resp.json());
        if (!result['success']) {
          throw new Error(t(`error.code_${result['errorCode']}`));
        }
      } catch (err) {
        console.error(err);
      }
    })();
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
        machineType={machineType}
        nextStep={handleNextStep}
        requestFocus={() => handleRequestFocus(stepIndex)}
        setMachineType={setMachineType}
        stepNum={stepIndex + 1}
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
        preferDetect={preferDetect}
        requestFocus={() => handleRequestFocus(stepIndex)}
        serverVersion={serverVersion}
        setPreferDetect={setPreferDetect}
        setServerVersion={setServerVersion}
        stepNum={stepIndex + 1}
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
        setWorldSource={xSetWorldSource}
        stepNum={stepIndex + 1}
        worldSource={worldSource}
      />
    );
  }

  if (worldSource === WorldLocation.Backups) {
    {
      const stepIndex = configItems.length;
      configItems.push(
        <ChooseBackup
          key="chooseBackup"
          backupGeneration={backupGeneration}
          isFocused={currentStep === stepIndex}
          nextStep={handleNextStep}
          requestFocus={() => handleRequestFocus(stepIndex)}
          setBackupGeneration={setBackupGeneration}
          setWorldName={setWorldName}
          stepNum={stepIndex + 1}
          worldName={worldName}
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
          setWorldName={setWorldName}
          stepNum={stepIndex + 1}
          worldName={worldName}
        />
      );
    }
    {
      const stepIndex = configItems.length;
      configItems.push(
        <ConfigureWorld
          key="configureWorld"
          isFocused={currentStep === stepIndex}
          levelType={levelType}
          nextStep={handleNextStep}
          requestFocus={() => handleRequestFocus(stepIndex)}
          seed={seed}
          setLevelType={setLevelType}
          setSeed={setSeed}
          stepNum={stepIndex + 1}
        />
      );
    }
  }

  const stepCount = configItems.length;

  return (
    <Card sx={{p: 2, mt: 6}} variant="outlined">
      {configItems}
      <Box sx={{textAlign: 'end'}}>
        <Button disabled={currentStep !== stepCount} onClick={handleStart} startIcon={<StartIcon />} type="button" variant="contained">
          {t('launch_server')}
        </Button>
      </Box>
    </Card>
  );
};

export default ServerConfigPane;
