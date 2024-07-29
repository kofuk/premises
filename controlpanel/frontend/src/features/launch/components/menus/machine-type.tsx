import {useTranslation} from 'react-i18next';

import {MenuItem as MUIMenuItem, Select, SelectChangeEvent, Stack, Typography} from '@mui/material';

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

  const machineType = config.machineType || '4g';

  const handleChange = (event: SelectChangeEvent<string>) => {
    updateConfig({machineType: event.target.value});
  };

  return {
    title: t('launch.machine_type'),
    ui: (
      <Stack sx={{mt: 0.5}}>
        <Select onChange={handleChange} value={machineType}>
          {machines.map((e, i) => (
            <MUIMenuItem key={`${i}`} value={e.name}>
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
    detail: valueLabel(config.machineType, (machineType) => machines.find((e) => e.name == machineType)!.getLabel()),
    variant: 'dialog',
    cancellable: true
  };
};
