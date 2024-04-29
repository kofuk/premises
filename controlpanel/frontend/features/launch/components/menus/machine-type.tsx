import React, {useState} from 'react';

import {useTranslation} from 'react-i18next';

import {Box, Slider, Stack, Typography} from '@mui/material';

import {useLaunchConfig} from '../launch-config';
import {MenuItem} from '../menu-container';

import {valueLabel} from './common';

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
    return `${this.memSize}GB RAM, ${this.nCores}-core CPU, ¥${this.price}/h`;
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

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const [machineType, setMachineType] = useState(config.machineType || '4g');

  const handleChange = (event: Event, newValue: number | number[]) => {
    setMachineType(machines[newValue as number].name);
  };

  return {
    title: t('config_machine_type'),
    ui: (
      <Stack>
        <Slider
          marks={machines.map((e, i) => ({value: i, label: e.getMemSizeLabel()}))}
          max={machines.length - 1}
          min={0}
          onChange={handleChange}
          sx={{mt: 2}}
          value={machines.findIndex((e) => e.name == machineType)}
        />
        <Box sx={{textAlign: 'center', mt: 2}}>
          <Typography sx={{opacity: 0.8}}>{machines.find((e) => e.name == machineType)!.getLabel()}</Typography>
        </Box>
      </Stack>
    ),
    detail: valueLabel(config.machineType, (machineType) => machines.find((e) => e.name == machineType)!.getLabel()),
    variant: 'dialog',
    cancellable: true,
    action: {
      label: t('save'),
      callback: () => {
        updateConfig({machineType});
      }
    }
  };
};
