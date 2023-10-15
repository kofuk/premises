export const decodeBuffer = (value: string): Uint8Array => {
  const stdEncoding = value.replaceAll('-', '+').replaceAll('_', '/');
  console.log(stdEncoding);
  return Uint8Array.from(atob(stdEncoding), (c) => c.charCodeAt(0));
};

export const encodeBuffer = (value: ArrayBuffer): string => {
  const stdEncoding = btoa(String.fromCharCode.apply(null, new Uint8Array(value) as unknown as number[]));
  return stdEncoding.replaceAll('+', '-').replaceAll('/', '_').replaceAll('=', '');
};
