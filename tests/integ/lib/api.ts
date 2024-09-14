// deno-lint-ignore-file no-explicit-any

import codes from "./codes.ts";

const TARGET_HOST = Deno.env.get("TARGET_HOST")!;

type ApiResponse = {
  success: boolean;
  errorCode?: number;
  data?: any;
};

const request = async (
  methodAndPath: string,
  accessToken?: string | null,
  body?: URLSearchParams | any | null,
  options?: {
    accept: string;
  },
): Promise<Response> => {
  let bodyStr;

  if (body) {
    bodyStr = JSON.stringify(body);
  }

  const [method, path] = methodAndPath.split(" ");

  const headers = new Headers();
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }
  if (options?.accept) {
    headers.set("Accept", options.accept);
  }
  headers.set("Content-Type", "application/json");

  return await fetch(`${TARGET_HOST}${path}`, {
    method: method,
    headers,
    body: bodyStr,
  });
};

const api = async (
  methodAndPath: string,
  accessToken?: string,
  body?: URLSearchParams | any,
): Promise<any> => {
  const response: ApiResponse = await request(methodAndPath, accessToken, body)
    .then(
      (response) => response.json(),
    );

  if (!response.success) {
    throw new Error(
      `API error: ${methodAndPath}: ${codes.error(response.errorCode!)}`,
    );
  }

  return response.data!;
};
export default api;

export const login = async (
  userName: string,
  password: string,
): Promise<string> => {
  const rawResponse = await fetch(`${TARGET_HOST}/api/internal/login`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "Origin": TARGET_HOST,
    },
    body: JSON.stringify({ userName, password }),
  });
  const cookie = rawResponse.headers.get("Set-Cookie");
  if (cookie === null) {
    throw Error(`Session cookie is not set`);
  }

  const loginResponse = await rawResponse.json();
  if (!loginResponse.success) {
    throw Error(`Login error: ${codes.error(loginResponse.errorCode!)}`);
  }

  const sessionData = await fetch(`${TARGET_HOST}/api/internal/session-data`, {
    headers: {
      cookie,
    },
  }).then((response) => response.json());
  if (!sessionData.success) {
    throw Error(`Login failed: ${codes.error(loginResponse.errorCode!)}`);
  }

  return sessionData.data.accessToken;
};

export const streamEvent = async (
  path: string,
  accessToken: string,
): Promise<any> => {
  const response = await request(`GET ${path}`, accessToken, null, {
    accept: "application/json",
  }).then((response) => response.json());
  return response;
};

export const params = (params: Record<string, string>): URLSearchParams => {
  const result = new URLSearchParams();
  for (const key of Object.keys(params)) {
    result.set(key, params[key]);
  }
  return result;
};
