import {ReactNode} from 'react';
import {ItemProp} from './prop';
import StepTitle from './step-title';

export default ({
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
        <svg width="30" height="30" viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg" version="1.1">
          <circle cx="50" cy="50" r="50" fill={isFocused ? 'blue' : 'gray'} />
          <text x="50" y="45" textAnchor="middle" dominantBaseline="central" fontFamily="sans-serif" fontSize="50" fill="white">
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
