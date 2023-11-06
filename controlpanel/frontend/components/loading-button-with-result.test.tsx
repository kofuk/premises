import React from 'react';

import {render, screen} from '@testing-library/react';

import LoadingButtonWithResult from './loading-button-with-result';

describe('LoadingButtonWithResult', () => {
  it('renders state change', async () => {
    const {rerender} = render(<></>);

    for (let i = 0; i < 8; i++) {
      const disabled = ((i >> 0) & 1) == 1;
      const loading = ((i >> 1) & 1) == 1;
      const success = ((i >> 2) & 1) == 1;

      rerender(
        <LoadingButtonWithResult disabled={disabled} loading={loading} success={success} type="button">
          inner text
        </LoadingButtonWithResult>
      );

      if (loading) {
        expect(screen.getByRole('progressbar')).toBeVisible();
      }
      if (disabled || loading) {
        expect(screen.getByRole('button')).toBeDisabled();
      }
      if (!loading && success) {
        expect(screen.getByTestId('CheckIcon')).toBeVisible();
      }
      if (!loading && !success) {
        expect(screen.getByText('inner text')).toBeVisible();
      }
    }
  });
});
