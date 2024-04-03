import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Share as ShareIcon, PlayArrow as StartIcon} from '@mui/icons-material';
import {Box, Button, Card} from '@mui/material';

import {useLaunchConfig} from './components/launch-config';

import ChooseBackup from '@/features/launch/config-item/choose-backup';
import ConfigureWorld from '@/features/launch/config-item/configure-world';
import MachineType from '@/features/launch/config-item/machine-type';
import ServerVersion from '@/features/launch/config-item/server-version';
import WorldName from '@/features/launch/config-item/world-name';
import WorldSource, {WorldLocation} from '@/features/launch/config-item/world-source';

const ServerConfigPane = () => {
  const [t] = useTranslation();

  const [worldSource, setWorldSource] = useState(WorldLocation.Backups);
  const [currentStep, setCurrentStep] = useState(0);

  const {launch, configId} = useLaunchConfig();

  const handleStart = () => {
    (async () => {
      try {
        await launch();
      } catch (err) {
        console.error(err);
      }
    })();
  };

  const handleShareConfig = () => {
    const url = new URL(location.href);
    url.hash = '#configShareId=' + configId;
    navigator.clipboard.writeText(url.toString());
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
        setWorldSource={setWorldSource}
        stepNum={stepIndex + 1}
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
        />
      );
    }
  }

  const stepCount = configItems.length;

  return (
    <Card sx={{p: 2, mt: 6}} variant="outlined">
      {configItems}
      <Box sx={{textAlign: 'end'}}>
        <Button disabled={currentStep !== stepCount} onClick={handleShareConfig} startIcon={<ShareIcon />} type="button" variant="outlined">
          {t('share_config')}
        </Button>
        <Button disabled={currentStep !== stepCount} onClick={handleStart} startIcon={<StartIcon />} sx={{mx: 1}} type="button" variant="contained">
          {t('launch_server')}
        </Button>
      </Box>
    </Card>
  );
};

export default ServerConfigPane;
