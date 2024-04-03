import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {ArrowDownward as NextIcon} from '@mui/icons-material';
import {Box, Button, Slider} from '@mui/material';

import {useLaunchConfig} from '../components/launch-config';

import ConfigContainer from '@/features/launch/config-item/config-container';
import {ItemProp} from '@/features/launch/config-item/prop';

class Machine {
  name: string;
  memSize: number;
  nCores: number;
  price: number;

  constructor(name: string, memSize: number, nCores: number, price: number) {
    this.name = name;
    this.memSize = memSize;
    this.nCores = nCores;
    this.price = price;
  }

  getLabel = (): string => {
    return `${this.memSize}GB RAM & ${this.nCores}-core CPU, ¥${this.price}/h`;
  };

  getMemSizeLabel = (): string => {
    return `${this.memSize} GB`;
  };
}

const machines: Machine[] = [
  new Machine('2g', 2, 3, 3.3),
  new Machine('4g', 4, 4, 6.6),
  new Machine('8g', 8, 6, 13.2),
  new Machine('16g', 16, 8, 24.2),
  new Machine('32g', 32, 12, 48),
  new Machine('64g', 64, 24, 96.8)
];

const MachineType = ({isFocused, nextStep, requestFocus, stepNum}: ItemProp) => {
  const [t] = useTranslation();

  const {updateConfig, config} = useLaunchConfig();

  const [machineType, setMachineType] = useState(config.machineType || '4g');

  const handleChange = (event: Event, newValue: number | number[]) => {
    setMachineType(machines[newValue as number].name);
  };

  const saveAndContinue = () => {
    (async () => {
      await updateConfig({machineType});
      nextStep();
    })();
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_machine_type')}>
      <Box sx={{mx: 5}}>
        <Slider
          marks={machines.map((e, i) => ({value: i, label: e.getMemSizeLabel()}))}
          max={machines.length - 1}
          min={0}
          onChange={handleChange}
          value={machines.findIndex((e) => e.name == machineType)}
          valueLabelDisplay="auto"
          valueLabelFormat={(i) => machines[i].getLabel()}
        />
      </Box>
      <Box sx={{textAlign: 'end'}}>
        <Button endIcon={<NextIcon />} onClick={saveAndContinue} type="button" variant="outlined">
          {t('next')}
        </Button>
      </Box>
    </ConfigContainer>
  );
};

export default MachineType;
