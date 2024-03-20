import codes from "../lib/codes.ts";
import api, { login, params, streamEvent } from "../lib/api.ts";

console.log("Login");
const cookie = await login("user1", "password1");

console.log("Launch server");
await api(
  "POST /api/launch",
  cookie,
  params({
    "machine-type": "2g",
    "server-version": "1.20.4",
    "prefer-detect": "true",
    "world-source": "new-world",
    "world-name": `test-${Math.random()}`,
    "seed": "",
    "level-type": "default",
  }),
);

console.log("Wait server launched");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await streamEvent("/api/streaming/events", cookie);
  if (pageCode === codes.PAGE.RUNNING) {
    break;
  } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
    throw new Error("Unexpected page");
  }
}

console.log("Stop server");
await api("POST /api/stop", cookie);

console.log("Wait server stopped");
for (let i = 0; i < 18; i++) {
  console.log("...");
  await new Promise((resolve) => setTimeout(resolve, 10 * 1000));

  const { pageCode } = await streamEvent("/api/streaming/events", cookie);
  if (pageCode === codes.PAGE.LAUNCH) {
    break;
  } else if (pageCode === codes.PAGE.MANUAL_SETUP) {
    throw new Error("Unexpected page");
  }
}

console.log("Success");
