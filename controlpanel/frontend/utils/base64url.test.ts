import * as base64url from './base64url';

describe('base64url', () => {
  describe('decode', () => {
    it('returns decoded buffer', () => {
      const result = base64url.decodeBuffer('A_-_-A');
      expect(result).toEqual(new Uint8Array([0x03, 0xff, 0xbf, 0xf8]));
    });
  });

  describe('encode', () => {
    it('returns URL safe Base64 encoded string', () => {
      const buffer = new ArrayBuffer(4);
      const source = new Uint8Array(buffer);
      source.set([0x03, 0xff, 0xbf, 0xf8]);
      const result = base64url.encodeBuffer(source);
      expect(result).toEqual('A_-_-A');
    });
  });
});
