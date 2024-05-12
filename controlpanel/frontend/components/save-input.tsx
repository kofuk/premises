import React, {FormEvent, useEffect, useState} from 'react';

import {Save as SaveIcon} from '@mui/icons-material';
import {FormControl, IconButton, InputAdornment, InputLabel, OutlinedInput} from '@mui/material';

type Props = {
  label: string;
  initValue: string;
  fullWidth?: boolean;
  type: string;
  onSave?: (value: string) => void;
  unsuitableForPasswordAutoFill?: boolean;
};

const SaveInput = ({label, initValue, fullWidth, type, onSave, unsuitableForPasswordAutoFill}: Props) => {
  const [value, setValue] = useState(initValue);

  useEffect(() => {
    setValue(initValue);
  }, [initValue]);

  const inputProps: any = {};
  if (unsuitableForPasswordAutoFill) {
    inputProps['data-1p-ignore'] = '';
  }

  const handleSave = (e: FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    onSave && onSave(value);
  };

  return (
    <form onSubmit={handleSave}>
      <FormControl fullWidth={fullWidth}>
        <InputLabel>{label}</InputLabel>
        <OutlinedInput
          endAdornment={
            <InputAdornment position="end">
              <IconButton disabled={value == initValue} type="submit">
                <SaveIcon color={value == initValue ? 'disabled' : 'primary'} />
              </IconButton>
            </InputAdornment>
          }
          inputProps={inputProps}
          label={label}
          onChange={(e) => setValue(e.target.value)}
          type={type}
          value={value}
        />
      </FormControl>
    </form>
  );
};

export default SaveInput;
