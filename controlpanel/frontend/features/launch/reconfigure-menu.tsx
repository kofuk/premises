import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Box, Button} from '@mui/material';

import {useLaunchConfig} from './components/launch-config';

import ChooseBackup from '@/features/launch/config-item/choose-backup';
import ConfigureWorld from '@/features/launch/config-item/configure-world';
import ServerVersion from '@/features/launch/config-item/server-version';
import WorldName from '@/features/launch/config-item/world-name';
import WorldSource, {WorldLocation} from '@/features/launch/config-item/world-source';

const ReconfigureMenu = () => {
  const [t] = useTranslation();

  const [worldSource, setWorldSource] = useState(WorldLocation.Backups);
  const [currentStep, setCurrentStep] = useState(0);

  const {reconfigure} = useLaunchConfig();

  const handleStart = () => {
    (async () => {
      try {
        await reconfigure();
      } catch (err) {
        console.error(err);
      }
    })();
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
    <Box>
      {configItems}
      <Box sx={{textAlign: 'end'}}>
        <Button disabled={currentStep !== stepCount} onClick={handleStart} type="button" variant="contained">
          {t('relaunch_server')}
        </Button>
      </Box>
    </Box>
  );
};

export default ReconfigureMenu;
