import styled from '@emotion/styled';

export default styled.h3`
  cursor: pointer;
  border-radius: 5px;
  transition: background-color 300ms;

  &:hover {
    background-color: rgba(0, 0, 0, 0.1);
  }
  &:active {
    background-color: rgba(0, 0, 0, 0.2);
  }
`;
