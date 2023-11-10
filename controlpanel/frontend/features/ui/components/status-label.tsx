import React from 'react';

import styled from 'styled-components';

type Prop = {
  message: string;
  progress: number;
};

const StatusContainer = styled.div<{progress: number}>`
  /*background-color: #93c0f5;*/
  background: ${(props) => 'linear-gradient(90deg, #6aa5eb 0%, #6aa5eb ' + props.progress + '%, #93c0f5 ' + props.progress + '%, #93c0f5 100%)'};
  color: black;
  width: 500px;
  padding: 5px 30px;
  border-radius: 1000px;
  border: solid 1px #99c1f0;
`;

const StatusLabel = ({message, progress}: Prop) => {
  return <StatusContainer progress={progress}>{message}</StatusContainer>;
};

export default StatusLabel;
