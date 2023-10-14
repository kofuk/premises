import React from 'react';

type Prop = {
  isError: boolean;
  message: string;
};

const StatusBar = ({isError, message}: Prop) => {
  const appearance = isError ? 'alert-danger' : 'alert-success';
  return <div className={`alert ${appearance}`}>{message}</div>;
};

export default StatusBar;
