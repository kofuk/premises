export const decodeBuffer = (value: string): Uint8Array => {
  return Uint8Array.from(atob(value), (c) => c.charCodeAt(0));
};

export const encodeBuffer = (value: ArrayBuffer): string => {
  return btoa(String.fromCharCode.apply(null, new Uint8Array(value) as unknown as number[]))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+/g, '');
};
