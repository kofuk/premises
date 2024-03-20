import codes from "../lib/codes.ts";

const TARGET_HOST = Deno.env.get("TARGET_HOST")!;

type ApiResponse = {
  success: boolean;
  errorCode?: number;
  data?: any;
};

const request = async (
  methodAndPath: string,
  cookie?: string | null,
  body?: any | URLSearchParams | null,
  options?: {
    accept: string;
  },
): Promise<Response> => {
  let bodyStr;

  let contentType = "application/json";
  if (body && body instanceof URLSearchParams) {
    contentType = "application/x-www-form-urlencoded";
    bodyStr = body.toString();
  } else if (body) {
    bodyStr = JSON.stringify(body);
  }

  const [method, path] = methodAndPath.split(" ");

  const headers = new Headers();
  if (cookie) {
    headers.set("Cookie", cookie);
  }
  if (options?.accept) {
    headers.set("Accept", options.accept);
  }
  headers.set("Origin", TARGET_HOST);
  headers.set("Content-Type", contentType);

  return await fetch(`${TARGET_HOST}${path}`, {
    method: method,
    headers,
    body: bodyStr,
  });
};

const api = async (
  methodAndPath: string,
  cookie?: string,
  body?: any|URLSearchParams,
): Promise<any> => {
  const response = await request(methodAndPath, cookie, body).then((
    response,
  ) => response.json());

  if (!response.success) {
    throw new Error(
      `API error: ${methodAndPath}: ${codes.error(response.errorCode)}`,
    );
  }

  return response.data!;
};
export default api;

export const login = async (
  userName: string,
  password: string,
): Promise<string> => {
  const rawResponse = await request(
    "POST /login",
    null,
    { userName, password },
  );

  const response: ApiResponse = await rawResponse.json();
  if (!response.success) {
    throw Error(`Login error: ${codes.error(response.errorCode!)}`);
  }

  const cookie = rawResponse.headers.get("Set-Cookie");
  if (cookie === null) {
    throw Error(`Session cookie is not set`);
  }

  return cookie;
};

export const streamEvent = async (
  path: string,
  cookie: string,
): Promise<any> => {
  const response = await request(`GET ${path}`, cookie, null, {
    accept: "application/ld+json",
  }).then((response) => response.json());
  return response;
};

export const params = (params: any): URLSearchParams => {
  const result = new URLSearchParams();
  for (const key of Object.keys(params)) {
    result.set(key, params[key]);
  }
  return result;
};
