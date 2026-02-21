import {MenuItem as MUIMenuItem, Select, type SelectChangeEvent, Stack, Typography} from '@mui/material';
import {useTranslation} from 'react-i18next';

import {useLaunchConfig} from '../launch-config';
import type {MenuItem} from '../menu-container';

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
  new Machine('2g', 2, 3, 3.7),
  new Machine('4g', 4, 4, 7.3),
  new Machine('12g', 12, 6, 14.6),
  new Machine('24g', 24, 8, 26.7),
  new Machine('48g', 48, 12, 53.3),
  new Machine('96g', 96, 24, 106.5),
  new Machine('128g', 128, 40, 142.8)
];

export const create = (): MenuItem => {
  const [t] = useTranslation();
  const {config, updateConfig} = useLaunchConfig();

  const machineType = config.machineType || '4g';

  const handleChange = (event: SelectChangeEvent<string>) => {
    updateConfig({machineType: event.target.value});
  };

  return {
    title: t('launch.machine_type'),
    ui: (
      <Stack sx={{mt: 0.5}}>
        <Select onChange={handleChange} value={machineType}>
          {machines.map((e) => (
            <MUIMenuItem key={e.name} value={e.name}>
              <Typography component="div" sx={{fontWeight: 400}} variant="body1">
                {e.getMemSizeLabel()}
              </Typography>
              <Typography component="div" sx={{mt: '3px', ml: 1, opacity: 0.7}} variant="body2">
                {e.getLabel()}
              </Typography>
            </MUIMenuItem>
          ))}
        </Select>
      </Stack>
    ),
    detail: valueLabel(config.machineType, (machineType) => machines.find((e) => e.name === machineType)!.getLabel()),
    variant: 'dialog',
    cancellable: true
  };
};
