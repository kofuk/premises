import React from 'react';

import {render} from '@testing-library/react';
import nock from 'nock';
import fetch from 'node-fetch';

import {AuthProvider} from '../auth';

const InnerComponent = () => {
  return <div>Test</div>;
};

describe('AuthProvider', () => {
  beforeEach(() => {
    location.replace(`http://localhost`);
    global.fetch = fetch as any as typeof global.fetch;
  });

  it('renders component after initialized', () => {
    nock('http://localhost').get('/api/current-user').reply(200, {
      success: true,
      user_name: 'hoge'
    });

    const component = render(
      <AuthProvider>
        <InnerComponent />
      </AuthProvider>
    );
    const tree = component.container;
    expect(tree).toMatchSnapshot();
  });
});
