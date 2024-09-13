import nock from 'nock';
import fetch from 'node-fetch';

import api, {APIError} from '.';

describe('api', () => {
  beforeEach(() => {
    global.fetch = fetch as any as typeof global.fetch;
  });

  it('sends and receive json value', async () => {
    nock('http://localhost')
      .post('/api/test', {foo: 'foo'})
      .reply(200, {
        success: true,
        data: {
          bar: 'bar'
        }
      });

    type Req = {
      foo: string;
    };

    type Resp = {
      bar: string;
    };

    const resp = await api<Req, Resp>('/api/test', 'post', 'xxxx', {foo: 'foo'});
    expect(resp.bar).toEqual('bar');
  });

  it('throws error if failed', async () => {
    nock('http://localhost').get('/api/test').reply(200, {
      success: false,
      errorCode: 1,
      reason: 'test'
    });

    expect(api<null, null>('/api/test', 'xxxx', 'get')).rejects.toThrow(APIError);
  });
});
