import React, {ReactNode} from 'react';

import {ItemProp} from './prop';
import StepTitle from './step-title';

const ConfigContainer = ({
  title,
  isFocused,
  requestFocus,
  stepNum,
  children
}: ItemProp & {
  title: string;
  children: ReactNode;
}) => {
  return (
    <div className="d-flex flex-row m-2">
      <div className="my-2">
        <svg height="30" version="1.1" viewBox="0 0 100 100" width="30" xmlns="http://www.w3.org/2000/svg">
          <circle cx="50" cy="50" fill={isFocused ? 'blue' : 'gray'} r="50" />
          <text dominantBaseline="central" fill="white" fontFamily="sans-serif" fontSize="50" textAnchor="middle" x="50" y="45">
            {stepNum}
          </text>
        </svg>
      </div>
      <div className="mx-2 p-2 flex-fill border rounded">
        <StepTitle className="user-select-none" onClick={requestFocus}>
          {title}
        </StepTitle>
        {isFocused && children}
      </div>
    </div>
  );
};

export default ConfigContainer;
