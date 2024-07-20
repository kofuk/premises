import {render, screen, waitFor} from '@testing-library/react';
import nock from 'nock';

import {AuthProvider, useAuth} from './auth';
import '@testing-library/jest-dom';

describe('AuthProvider', () => {
  it('renders component after initialized', async () => {
    nock('http://localhost')
      .get('/api/session-data')
      .reply(200, {
        success: true,
        data: {
          loggedIn: true,
          userName: 'hoge'
        }
      });

    const InnerComponent = () => {
      const {loggedIn, userName} = useAuth();
      return (
        <div>
          <h1>MAIN</h1>
          <span>{`loggedIn=${loggedIn}`}</span>
          <span>{`userName=${userName}`}</span>
        </div>
      );
    };

    render(
      <AuthProvider>
        <InnerComponent />
      </AuthProvider>
    );

    expect(screen.queryByText('MAIN')).not.toBeInTheDocument();

    await waitFor(() => expect(screen.getByText('MAIN')).toBeInTheDocument());

    expect(screen.getByText('loggedIn=true')).toBeInTheDocument();
    expect(screen.getByText('userName=hoge')).toBeInTheDocument();
  });
});
