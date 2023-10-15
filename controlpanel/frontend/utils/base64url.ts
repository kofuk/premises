export const decodeBuffer = (value: string): Uint8Array => {
  const stdEncoding = value.replace(/-/g, '+').replace(/_/g, '/');
  return Uint8Array.from(atob(stdEncoding), (c) => c.charCodeAt(0));
};

export const encodeBuffer = (value: ArrayBuffer): string => {
  const stdEncoding = btoa(String.fromCharCode.apply(null, new Uint8Array(value) as unknown as number[]));
  return stdEncoding.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+/g, '');
};
