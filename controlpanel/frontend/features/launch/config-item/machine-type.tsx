import React from 'react';

import {useTranslation} from 'react-i18next';

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

  createReactElement = (selectedValue: string, clickHandler: (val: string) => void): React.ReactElement => {
    return (
      <React.Fragment key={this.name}>
        <input
          autoComplete="off"
          checked={this.name === selectedValue}
          className="btn-check"
          id={`machineType_${this.name}`}
          name="machine-type"
          onChange={() => clickHandler(this.name)}
          type="radio"
          value={this.name}
        />
        <label className="btn btn-outline-primary" htmlFor={`machineType_${this.name}`} title={this.getLabel()}>
          {this.memSize} GB
        </label>
      </React.Fragment>
    );
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

const MachineType = ({
  isFocused,
  nextStep,
  requestFocus,
  stepNum,
  machineType,
  setMachineType
}: ItemProp & {
  machineType: string;
  setMachineType: (val: string) => void;
}) => {
  const [t] = useTranslation();

  const handleClick = (val: string) => {
    setMachineType(val);
  };

  return (
    <ConfigContainer isFocused={isFocused} nextStep={nextStep} requestFocus={requestFocus} stepNum={stepNum} title={t('config_machine_type')}>
      <div className="btn-group ms-3" role="group">
        {machines.map((e) => e.createReactElement(machineType, handleClick))}
      </div>
      <div className="m-1 text-end">
        <button className="btn btn-primary" onClick={nextStep} type="button">
          {t('next')}
        </button>
      </div>
    </ConfigContainer>
  );
};

export default MachineType;
