import React from 'react';

type Prop = {
  message: string;
};

const StatusBar = ({message}: Prop) => {
  return <div className={`alert alert-success`}>{message}</div>;
};

export default StatusBar;
